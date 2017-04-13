package kubeadm

import (
	"path"
	"strconv"

	kubeadmapi "k8s.io/kubernetes/cmd/kubeadm/app/apis/kubeadm"
	kubeadmconstants "k8s.io/kubernetes/cmd/kubeadm/app/constants"
	kubemaster "k8s.io/kubernetes/cmd/kubeadm/app/master"
	addonsphase "k8s.io/kubernetes/cmd/kubeadm/app/phases/addons"
	apiconfigphase "k8s.io/kubernetes/cmd/kubeadm/app/phases/apiconfig"
)

func Addons(kmmCfg Config) error {

	adminKubeConfigPath := path.Join(kubeadmapi.GlobalEnvParams.KubernetesDir, kubeadmconstants.AdminKubeConfigFileName)
	client, err := kubemaster.CreateClientAndWaitForAPI(adminKubeConfigPath)
	if err != nil {
		return err
	}

	if err := apiconfigphase.UpdateMasterRoleLabelsAndTaints(client); err != nil {
		return err
	}

	// PHASE 5: Install and deploy all addons, and configure things as necessary

	// Create the necessary ServiceAccounts
	err = apiconfigphase.CreateServiceAccounts(client)
	if err != nil {
		return err
	}

	err = apiconfigphase.CreateRBACRules(client)
	if err != nil {
		return err
	}

	var cfg *kubeadmapi.MasterConfiguration
	if cfg, err = GetKubeadmCfg(kmmCfg); err != nil {
		return err
	}

	if err := addonsphase.CreateEssentialAddons(cfg, client); err != nil {
		return err
	}
	return nil
}

// TODO: This is a hack until we can use kubeadm cmd directly...
func GetKubeadmCfg(kmmCfg Config) (*kubeadmapi.MasterConfiguration, error) {
	var cfg = &kubeadmapi.MasterConfiguration{}
	cfg.API.AdvertiseAddress = kmmCfg.ApiServer.String()

	// TODO: get this from cmd or Tags...
	cfg.KubernetesVersion = "v1.6.1"

	port := kmmCfg.ApiServer.Port()
	if port == "" {
		cfg.API.BindPort = 6443
	} else {
		// Parse the port
		var i64 int64
		var err error
		if i64, err = strconv.ParseInt(port, 10, 32); err != nil {
			return cfg, err
		}
		cfg.API.BindPort = int32(i64)
	}
	return cfg, nil
}
