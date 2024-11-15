#!/usr/bin/env bash
set -eu -o pipefail

PROJECT_ROOT="$(git rev-parse --show-toplevel)"
declare -r PROJECT_ROOT
declare -r BACKUP_DIR=${BACKUP_DIR:-"$PROJECT_ROOT/tmp/grafana-deployment/$(date +%Y-%m-%d-%H-%M-%S)"}

declare -r CMO_ENABLE_UWM_CFG=hack/dashboard/openshift/uwm/00-openshift-monitoring-user-projects.yaml
declare -r MON_NS=openshift-monitoring
declare -r UWM_NS=openshift-user-workload-monitoring
declare -r CMO_CM=cluster-monitoring-config
declare -r BACKUP_CMO_CFG="$BACKUP_DIR/cmo-cm.yaml"
declare -r UWM_URL="https://docs.openshift.com/container-platform/latest/observability/monitoring/enabling-monitoring-for-user-defined-projects.html"
declare -r UWM_CONFIG_URL="https://docs.openshift.com/container-platform/latest/observability/monitoring/configuring-the-monitoring-stack.html#configuring-the-monitoring-stack_configuring-the-monitoring-stack"

declare -r GRAFANA_NS=kepler-grafana
declare -r GRAFANA_SA=grafana

source "$PROJECT_ROOT/hack/utils.bash"

oc_apply_grafana_ns() {
	run oc apply -n "$GRAFANA_NS" "$@"
}

oc_replace_grafana_ns() {
	oc replace --force -n "$GRAFANA_NS" "$@"
}

oc_get_grafana_ns() {
	oc get -n "$GRAFANA_NS" "$@"
}

oc_delete_grafana_ns() {
	run oc -n "$GRAFANA_NS" delete --ignore-not-found=true "$@"
}

oc_create_grafana_ns() {
	run oc -n "$GRAFANA_NS" create "$@"
}

validate_cluster() {
	header "Validating cluster"

	command -v oc >/dev/null 2>&1 || {
		fail "No oc command found in PATH"
		info "Please install oc"
		cat <<-EOF
			curl -sNL https://mirror.openshift.com/pub/openshift-v4/x86_64/clients/ocp/4.16.0/openshift-client-linux.tar.gz |
			  tar -xzf - -C <install/path>
		EOF
		# NOTE: do not proceed without oc installed
		return 1
	}

	local -i ret=0
	local oc_version=""
	oc_version=$(oc version --client -oyaml | grep releaseClientVersion: | cut -f2 -d:)
	[[ -z "$oc_version" ]] && {
		oc_version=$(oc version --client -oyaml | grep gitVersion | cut -f2 -d:)
	}

	local -i oc_major_version oc_minor_version
	oc_major_version=$(echo "$oc_version" | cut -f1 -d.)
	oc_minor_version=$(echo "$oc_version" | cut -f2 -d.)
	info "Found oc version: $oc_version -> ($oc_major_version.$oc_minor_version.z)"

	[[ $oc_major_version -lt 4 ]] || [[ $oc_minor_version -lt 12 ]] && {
		fail "oc version '$oc_version' should be at least 4.12.0"
		info "install a newer version of oc"
		cat <<-EOF
			curl -sNL https://mirror.openshift.com/pub/openshift-v4/x86_64/clients/ocp/4.13.0/openshift-client-linux.tar.gz |
			  tar -xzf - -C <install/path>
		EOF
		ret=1
	}

	oc get ns $MON_NS -o name >/dev/null 2>&1 || {
		fail "$MON_NS namespace missing. Is this an OpenShift cluster?"
		info "cluster:  $(oc whoami --show-server)\n"
		ret=1
	}

	oc get crds keplers.kepler.system.sustainable.computing.io || {
		fail "Missing Kepler CRD. Is Kepler Operator Installed?\n"
		ret=1
	}

	oc get kepler kepler >/dev/null 2>&1 || {
		fail "Missing kepler instance. Did you forget to create kepler?"
		info_run "oc apply -f ./config/samples/kepler.system_v1alpha1_kepler.yaml"
		ret=1
	}

	return $ret
}

wait_for_kepler_to_be_available() {
	header "waiting for kepler to be available"
	wait_until 10 10 "kepler to be available" condition_check "True" oc get kepler kepler \
		-o jsonpath="{.status.exporter.conditions[?(@.type=='Available')].status}" && {
		ok "kepler is available"
		return 0
	}
	fail "kepler not ready"
	run oc get kepler
	line 50
	oc get kepler kepler -o jsonpath="$(
		cat <<-EOF
			{range .status.exporter.conditions[?(@.status!="True")]}
				{" * "}{.type}{":"} {.status}
				      {.reason}
				      {.message}
			{end}
		EOF
	)"
	line 50
	echo
	info "Please check the operator logs for more details."
	info_run "oc logs -n openshift-operators deployment/kepler-operator-controller"
	return 1
}

enable_userworkload_monitoring() {
	header "Enabling Openshift User Project Monitoring"

	info "checking if user project monitoring is already enabled"
	[[ $(oc get prometheus -n $UWM_NS -o name | wc -l) -gt 0 ]] && {
		ok "User project monitoring already enabled; no further action is required"
		return 0
	}

	backup_monitoring_config

	if [[ -f "$BACKUP_CMO_CFG" ]]; then
		patch_enable_uwm
	else
		setup_new_uwm
	fi
	show_restore_info

	wait_until 10 10 "$UWM_NS to be created" oc get ns "$UWM_NS"
	wait_until 10 10 "User Workload Prometheus to be created" \
		oc wait --for condition=Available -n "$UWM_NS" prometheus user-workload
}

patch_enable_uwm() {
	info "Patching existing config to enable User Project Monitoring"
	line 60

	if grep enableUserWorkload "$BACKUP_CMO_CFG"; then
		info "toggling enableUserWorkload to true"
		sed -e 's|enableUserWorkload:.*|enableUserWorkload: true 👈 |g' <"$BACKUP_CMO_CFG"

		sed -e 's|enableUserWorkload:.*|enableUserWorkload: true|g' <"$BACKUP_CMO_CFG" |
			oc apply --server-side --force-conflicts -f-

	else
		info "adding enableUserWorkload to true"
		sed \
			-e 's|\(config.yaml.*\)|\1\n    enableUserWorkload: true 👈\r|g' \
			"$BACKUP_CMO_CFG"

		sed \
			-e 's|\(config.yaml.*\)|\1\n    enableUserWorkload: true\r|g' \
			"$BACKUP_CMO_CFG" |
			oc apply --server-side --force-conflicts -f-
	fi

}

setup_new_uwm() {
	info "Creating new config to enable User Project Monitoring"
	line 60
	cat $CMO_ENABLE_UWM_CFG
	line 60
	run oc apply -n "$MON_NS" -f "$CMO_ENABLE_UWM_CFG"
}

show_restore_info() {
	line 60 heavy
	warn "😱 In the event cluster-monitoring-operator becomes 👉 degraded  😱"
	cat <<-EOF

		  * Restore the configuration $BACKUP_CMO_CFG 
		  * Manually enable User Project Monitoring 
			  💡 see: $UWM_URL
		  * Rerun this script

	EOF
	line 60 heavy
	sleep 10
}

show_uwm_info() {
	info "Kepler use prometheus deployed in $UWM_NS to store metrics. To configure Prometheus to cater to needs of the cluster such as:"
	cat <<-EOF

		    * Increase data retention for in-depth analysis
		    * Allocate more resources based on requirements

		💡 see: $UWM_CONFIG_URL

	EOF
	line 55 heavy
}

backup_monitoring_config() {
	info "Backing up current monitoring configuration"

	oc get -n "$MON_NS" configmap "$CMO_CM" || {
		info "No monitoring config exists; will create a new one"
		return
	}

	# backup but remove last-applied-configuration
	oc get -n "$MON_NS" configmap "$CMO_CM" -o yaml |
		grep -Ev 'kubectl.kubernetes.io/last-applied-configuration|{"apiVersion":|creationTimestamp:|resourceVersion:|uid:' >"$BACKUP_CMO_CFG" || {
		fail "failed to backup $CMO_CM in $MON_NS"
		return 1
	}

	ok "Backed up $CMO_CM config to $BACKUP_DIR"
}

setup_grafana() {
	header "Setting up Grafana"

	info "Creating Grafana resources"

	# NOTE: create | apply will create only if needed
	info "Creating grafana namespace - $GRAFANA_NS"
	oc create ns $GRAFANA_NS --dry-run=client -o yaml | oc apply -f -
	oc_apply_grafana_ns -k hack/dashboard/openshift/grafana-deploy

	wait_until 20 10 "Grafana to be deployed" \
		oc get grafana,grafanadatasource

	wait_until 10 10 "Grafana CRDs to be Established" \
		oc wait --for=condition=Established crd grafanas.grafana.integreatly.org
	wait_until 10 10 "Grafana Datasource CRDs to be Established" \
		oc wait --for=condition=Established crd grafanadatasources.grafana.integreatly.org
	ok "grafana crds created\n"

	info "Creating a grafana instance"
	oc_apply_grafana_ns -f hack/dashboard/openshift/grafana-config/01-grafana-instance.yaml
	ok "Grafana created"
}

config_grafana_sa() {
	header "Grafana User Project Monitoring Setup"

	oc_apply_grafana_ns -f hack/dashboard/openshift/grafana-config/02-grafana-sa.yaml
	wait_until 20 2 "Grafana Service Account" \
		oc_get_grafana_ns serviceaccount $GRAFANA_SA

	run oc adm policy add-cluster-role-to-user cluster-monitoring-view \
		-z $GRAFANA_SA -n "$GRAFANA_NS"

	ok "grafana $GRAFANA_SA added to cluster-monitoring-view"
}

setup_grafana_dashboard() {
	header "Setting up Grafana dashboard"

	info "Creating datasource"
	local sa_token=""
	# sa_token="$(oc -n "$GRAFANA_NS" create token "$GRAFANA_SA")"
	oc_create_grafana_ns token --duration=8760h "$GRAFANA_SA" >tmp/grafana-token
	sa_token=$(cat tmp/grafana-token)

	# Deploy from updated manifest
	BEARER_TOKEN="$sa_token" \
		envsubst <hack/dashboard/openshift/grafana-config/03-grafana-datasource-UPDATETHIS.yaml |
		oc apply -n "$GRAFANA_NS" -f -
	ok "created grafana datasource \n"

	info "Creating dashboard config map"
	oc_delete_grafana_ns configmap kepler-dashboard-cm
	oc_create_grafana_ns configmap kepler-dashboard-cm --from-file=hack/dashboard/assets/kepler/dashboard.json

	oc_delete_grafana_ns configmap prometheus-dashboard-cm
	oc_create_grafana_ns configmap prometheus-dashboard-cm --from-file=hack/dashboard/assets/prometheus/dashboard.json

	info "Creating Dashboard"
	oc_delete_grafana_ns -f hack/dashboard/openshift/grafana-config/04-kepler-dashboard.yaml
	oc_create_grafana_ns -f hack/dashboard/openshift/grafana-config/04-kepler-dashboard.yaml

	oc_delete_grafana_ns -f hack/dashboard/openshift/grafana-config/05-prometheus-dashboard.yaml
	oc_create_grafana_ns -f hack/dashboard/openshift/grafana-config/05-prometheus-dashboard.yaml

	# NOTE: route name is dependent on the grafana instance
	wait_until 20 2 "Grafana dashboard" \
		oc_get_grafana_ns route kepler-grafana-route

	ok "created grafana dashboard\n"
}

grafana_login_url() {
	local grafana_url=""
	echo "https://$(oc_get_grafana_ns route kepler-grafana-route -o jsonpath='{.spec.host}')/login"

}

show_key_info() {
	header "Grafana Dashboard Setup Complete"

	local grafana_url=""
	grafana_url=$(grafana_login_url)

	# disable use find instead of ls
	# shellcheck disable=SC2012
	[[ -d "$BACKUP_DIR" ]] && cat <<-EOF
		  📦 Cluster Monitoring Configuration 
			    Backup Directory: $BACKUP_DIR
			$(ls "$BACKUP_DIR" | sed -e "s|^|      • |g")
	EOF

	cat <<-EOF
		  📈 Grafana Configuration:

			   Dashboard URL: $grafana_url
			           Admin: kepler
			        Password: kepler
	EOF

	line 55 heavy
}

main() {
	cd "$PROJECT_ROOT"
	mkdir -p "$BACKUP_DIR"

	validate_cluster || {
		line 60 heavy
		fail "Cluster validation failed!"
		info "Fix issues reported above and rerun the script\n\n"
		return 1
	}
	ok "cluster validated"

	wait_for_kepler_to_be_available

	enable_userworkload_monitoring
	setup_grafana
	config_grafana_sa
	setup_grafana_dashboard
	# remove backup dir if no backups were made
	[[ -z "$(ls -A "$BACKUP_DIR")" ]] && rm -rf "$BACKUP_DIR"

	show_key_info
	show_uwm_info

}

main "$@"
