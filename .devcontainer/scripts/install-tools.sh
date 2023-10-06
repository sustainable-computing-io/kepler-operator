#!/bin/bash
######################################################################
# These scripts are meant to be run in user mode as they may modify
# usr settings line .bashrc and .bash_aliases
######################################################################

echo "**********************************************************************"
echo "Install OpenShift CLI..."
echo "**********************************************************************"
if [ "$(uname -m)" == aarch64 ]; then
	echo "Installing OpenShift CLI for ARM64..."
	curl https://mirror.openshift.com/pub/openshift-v4/clients/ocp/stable/openshift-client-linux-arm64.tar.gz --output oc.tar.gz
else
	echo "Installing OpenShift CLI for x86_64..."
	curl https://mirror.openshift.com/pub/openshift-v4/clients/ocp/stable/openshift-client-linux.tar.gz --output oc.tar.gz
fi
sudo tar xvzf oc.tar.gz -C /usr/local/bin/ oc
sudo ln -s /usr/local/bin/oc /usr/bin/oc
rm oc.tar.gz
oc version

echo "**********************************************************************"
echo "Creatine alias for kubectl (kc) and kustomize (ku)..."
echo "**********************************************************************"

echo "Creating kc and kns alias for kubectl..."
echo "alias kc='/usr/local/bin/kubectl'" >>"$HOME/.bash_aliases"
echo "alias kns='kubectl config set-context --current --namespace'" >>"$HOME/.bash_aliases"
echo "Creating ku alias for kustomize..."
echo "alias ku='/usr/local/bin/kustomize'" >>"$HOME/.bash_aliases"

echo "Extra tools installations complete."
