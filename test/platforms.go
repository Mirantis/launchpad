package test

type Platform struct {
	Name       string
	Count      int
	Type       string
	VolumeSize string
	Public     bool
	Role       string
	UserData   string
}

func (p Platform) GetManager() map[string]interface{} {
	return map[string]interface{}{
		"platform":    p.Name,
		"count":       p.Count,
		"type":        "m6a.2xlarge",
		"volume_size": p.VolumeSize,
		"public":      p.Public,
		"role":        "manager",
		"user_data":   p.UserData,
	}
}

func (p Platform) GetWorker() map[string]interface{} {
	return map[string]interface{}{
		"platform":    p.Name,
		"count":       p.Count,
		"type":        "c6a.xlarge",
		"volume_size": p.VolumeSize,
		"public":      p.Public,
		"role":        "worker",
		"user_data":   p.UserData,
	}
}

func (p Platform) GetMSR() map[string]interface{} {
	return map[string]interface{}{
		"platform":    p.Name,
		"count":       p.Count,
		"type":        "m6a.2xlarge",
		"volume_size": p.VolumeSize,
		"public":      p.Public,
		"role":        "msr",
		"user_data":   p.UserData,
	}
}

var Platforms = map[string]Platform{
	"Ubuntu20": {
		Name:       "ubuntu_20.04",
		Count:      1,
		VolumeSize: "100",
		Public:     true,
		UserData:   "sudo ufw allow 7946/tcp ; sudo ufw allow 10250/tcp ",
	},
	"Ubuntu22": {
		Name:       "ubuntu_22.04",
		Count:      1,
		VolumeSize: "100",
		Public:     true,
		UserData:   "sudo ufw allow 7946/tcp ; sudo ufw allow 10250/tcp ",
	},
	"Rhel9": {
		Name:       "rhel_9",
		Count:      1,
		VolumeSize: "100",
		Public:     true,
		UserData:   "sudo firewall-cmd --zone=public --add-port=7946/tcp --permanent ; sudo firewall-cmd --zone=public --add-port=10250/tcp --permanent ; sudo firewall-cmd --reload",
	},
	"Rhel8": {
		Name:       "rhel_8",
		Count:      1,
		VolumeSize: "100",
		Public:     true,
		UserData:   "sudo firewall-cmd --zone=public --add-port=7946/tcp --permanent ; sudo firewall-cmd --zone=public --add-port=10250/tcp --permanent ; sudo firewall-cmd --reload",
	},
	"Centos7": {
		Name:       "centos_7",
		Count:      1,
		VolumeSize: "100",
		Public:     true,
		UserData:   "sudo firewall-cmd --zone=public --add-port=7946/tcp --permanent ; sudo firewall-cmd --zone=public --add-port=10250/tcp --permanent ; sudo firewall-cmd --reload",
	},
	"Oracle9": {
		Name:       "oracle_9",
		Count:      1,
		VolumeSize: "100",
		Public:     true,
		UserData:   "sudo firewall-cmd --zone=public --add-port=7946/tcp --permanent ; sudo firewall-cmd --zone=public --add-port=10250/tcp --permanent ; sudo firewall-cmd --reload",
	},
	"Sles12": {
		Name:       "sles_12",
		Count:      1,
		VolumeSize: "100",
		Public:     true,
		UserData:   "",
	},
	"Sles15": {
		Name:       "sles_15",
		Count:      1,
		VolumeSize: "100",
		Public:     true,
		UserData:   "sudo firewall-cmd --zone=public --add-port=7946/tcp --permanent ; sudo firewall-cmd --zone=public --add-port=10250/tcp --permanent ; sudo firewall-cmd --reload",
	},
	"Rocky8": {
		Name:       "rocky_8",
		Count:      1,
		VolumeSize: "100",
		Public:     true,
		UserData:   "sudo firewall-cmd --zone=public --add-port=7946/tcp --permanent ; sudo firewall-cmd --zone=public --add-port=10250/tcp --permanent ; sudo firewall-cmd --reload",
	},
	"Rocky9": {
		Name:       "rocky_9",
		Count:      1,
		VolumeSize: "100",
		Public:     true,
		UserData:   "sudo firewall-cmd --zone=public --add-port=7946/tcp --permanent ; sudo firewall-cmd --zone=public --add-port=10250/tcp --permanent ; sudo firewall-cmd --reload",
	},
	"Windows2019": {
		Name:       "windows_2019",
		Count:      1,
		VolumeSize: "100",
		Public:     true,
		UserData:   "",
	},
	"Windows2022": {
		Name:       "windows_2022",
		Count:      1,
		VolumeSize: "100",
		Public:     true,
		UserData:   "",
	},
}
