package kubeadm

import (
	"fmt"
	"path"

	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	kubemaster "k8s.io/kubernetes/cmd/kubeadm/app/master"
	addonsphase "k8s.io/kubernetes/cmd/kubeadm/app/phases/addons"
	apiconfigphase "k8s.io/kubernetes/cmd/kubeadm/app/phases/apiconfig"
	"k8s.io/kubernetes/pkg/util/version"
)

// Addons - deploys the essential addons
func (k *Config) Addons() error {

	k8sVersion, err := version.ParseSemantic(k.KubeVersion)
	if err != nil {
		return fmt.Errorf("couldn't parse kubernetes version %q: %v", k.KubeVersion, err)
	}

	adminKubeConfigPath := path.Join(kubeadmapi.GlobalEnvParams.KubernetesDir, kubeadmconstants.AdminKubeConfigFileName)
	client, err := kubemaster.CreateClientAndWaitForAPI(adminKubeConfigPath)
	if err != nil {
		return err
	}

	if err := apiconfigphase.UpdateMasterRoleLabelsAndTaints(client, k.KubeletID); err != nil {
		return err
	}

	// PHASE 5: Install and deploy all addons, and configure things as necessary

	// Create the necessary ServiceAccounts
	err = apiconfigphase.CreateServiceAccounts(client)
	if err != nil {
		return err
	}

	err = apiconfigphase.CreateRBACRules(client, k8sVersion)
	if err != nil {
		return err
	}

	var kubeadmapiCfg *kubeadmapi.MasterConfiguration
	if kubeadmapiCfg, err = GetKubeadmCfg(*k); err != nil {
		return err
	}

	if err := addonsphase.CreateEssentialAddons(kubeadmapiCfg, client); err != nil {
		return err
	}
	return nil
}
