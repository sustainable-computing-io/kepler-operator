#!/usr/bin/env bash

set -eu -o pipefail

PROJECT_ROOT="$(git rev-parse --show-toplevel)"
declare -r PROJECT_ROOT
declare -r BACKUP_DIR=${BACKUP_DIR:-"$PROJECT_ROOT/tmp/grafana-deployment"}

declare -r CMO_ENABLE_UWM_CFG=hack/dashboard/openshift/uwm/00-openshift-monitoring-user-projects.yaml
declare -r MON_NS=openshift-monitoring
declare -r UWM_NS=openshift-user-workload-monitoring
declare -r CMO_CM=cluster-monitoring-config
declare -r BACKUP_CMO_CFG="$BACKUP_DIR/cmo-cm.yaml"

declare -r KEPLER_DEPLOYMENT_NS=openshift-kepler-operator
declare -r GRAFANA_SA=grafana

source "$PROJECT_ROOT/hack/utils.bash"

oc_apply_kepler_ns() {
	run oc apply -n "$KEPLER_DEPLOYMENT_NS" "$@"
}
oc_get_kepler_ns() {
	oc get -n "$KEPLER_DEPLOYMENT_NS" "$@"
}
oc_create_kepler_ns() {
	run oc -n "$KEPLER_DEPLOYMENT_NS" create "$@"
}

validate_cluster() {
	header "Validating cluster"

	command -v oc >/dev/null 2>&1 || {
		fail "No oc command found in PATH"
		info "Please install oc"
		cat <<-EOF
			curl -sNL https://mirror.openshift.com/pub/openshift-v4/x86_64/clients/ocp/4.13.0/openshift-client-linux.tar.gz |
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

	oc get ns $MON_NS -o name || {
		fail "$MON_NS namespace missing. Is this an OpenShift cluster?"
		info "cluster:  $(oc whoami --show-server)"
		ret=1
	}

	oc get crds | grep kepler || {
		fail "Missing Kepler CRD. Is Kepler Operator Installed?"
		ret=1
	}

	[[ $(oc get kepler kepler -o=jsonpath='{.status.conditions[?(@.type=="Available")].status}') == "True" ]] || {
		fail "Mising kepler deployment. Did you create kepler instance?"
		ret=1
	}

	return $ret
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

	wait_until 10 10 "$UWM_NS to be created " oc get ns "$UWM_NS"
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
			  💡 see: https://docs.openshift.com/container-platform/latest/monitoring/enabling-monitoring-for-user-defined-projects.html
		  * Rerun this script

	EOF
	line 60 heavy
	sleep 10
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
	oc_apply_kepler_ns -k hack/dashboard/openshift/grafana-deploy

	wait_until 20 10 "Grafana to be deployed" \
		oc get grafana,grafanadatasource

	wait_until 10 10 "Grafana CRDs to be Established" \
		oc wait --for=condition=Established crd grafanas.integreatly.org
	wait_until 10 10 "Grafana Datasource CRDs to be Established" \
		oc wait --for=condition=Established crd grafanadatasources.integreatly.org
	ok "grafana crds created\n"

	info "Creating a grafana instance"
	oc_apply_kepler_ns -f hack/dashboard/openshift/grafana-config/01-grafana-instance.yaml
	ok "Grafana created"
}

config_grafana_sa() {
	header "Grafana User Project Monitoring Setup"

	oc_apply_kepler_ns -f hack/dashboard/openshift/grafana-config/02-grafana-sa.yaml
	wait_until 20 2 "Grafana Service Account" \
		oc_get_kepler_ns serviceaccount $GRAFANA_SA

	run oc adm policy add-cluster-role-to-user cluster-monitoring-view \
		-z $GRAFANA_SA -n "$KEPLER_DEPLOYMENT_NS"

	ok "grafana $GRAFANA_SA added to cluster-monitoring-view"
}

setup_grafana_dashboard() {
	header "Setting up Grafana dashboard"

	info "Creating datasource"
	local sa_token=""
	# sa_token="$(oc -n "$KEPLER_DEPLOYMENT_NS" create token "$GRAFANA_SA")"
	oc -n "$KEPLER_DEPLOYMENT_NS" create token --duration=8760h "$GRAFANA_SA" >tmp/grafana-token
	sa_token=$(cat tmp/grafana-token)

	# Deploy from updated manifest
	BEARER_TOKEN="$sa_token" \
		envsubst <hack/dashboard/openshift/grafana-config/03-grafana-datasource-UPDATETHIS.yaml |
		oc apply -n "$KEPLER_DEPLOYMENT_NS" -f -
	ok "created grafana datasource \n"

	info "Creating dashboard config map"
	oc_create_kepler_ns configmap grafana-dashboard-cm --from-file=hack/dashboard/assets/dashboard.json

	info "Creating Dashboard"
	oc_apply_kepler_ns \
		-f hack/dashboard/openshift/grafana-config/04-grafana-dashboard.yaml

	# NOTE: route name is dependent on the grafana instance
	wait_until 20 2 "Grafana dashboard" \
		oc_get_kepler_ns route grafana-route

	ok "created grafana dashboard\n"
}

show_key_info() {
	header "Grafana Dashboard Setup Complete"

	local grafana_url=""
	grafana_url="https://$(oc_get_kepler_ns route grafana-route -o jsonpath='{.spec.host}')/login"

	# disable use find instead of ls
	# shellcheck disable=SC2012
	cat <<-EOF
		  📦 Cluster Monitoring Configuration 
			    Backup Directory: $BACKUP_DIR
			$(ls "$BACKUP_DIR" | sed -e "s|^|      • |g")

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

	enable_userworkload_monitoring
	setup_grafana
	config_grafana_sa
	setup_grafana_dashboard
	show_key_info

}

main "$@"
