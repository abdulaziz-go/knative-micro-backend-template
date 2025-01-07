package pkg

import (
	"crypto/tls"
	"net/http"
	"time"
)

var (
	Client               = &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
	MongoURL             = "mongodb://mbf_ebf657726f964d5fb08c65c915f85e2c_p_obj_build_svcs:kMiMD5V4VU@142.93.164.37:27017/mbf_ebf657726f964d5fb08c65c915f85e2c_p_obj_build_svcs"
	SingleURL            = "https://api.admin.u-code.io/v2/items/"
	GetListURL           = "https://api.admin.u-code.io/v2/object-slim/get-list/"
	MultipleUpdateUrl    = "https://api.admin.u-code.io/v1/object/multiple-update/"
	KnativeURL           = "http://mbf-sap-integration-service.knative-fn.u-code.io/"
	AppId                = "P-wlqGq6ckG4uTRuJZITbXxLzaLz6we0gk"
	SessionId            = ""
	RequestTimeout       = 30 * time.Second
	BusinessPartnerGroup = map[int]string{
		100: "Клиенты",
		101: "Поставщики",
		102: "Поставщики импорт",
		103: "Поставщики местные",
		104: "Поставщики услуг",
		105: "Таможня",
		106: "Транспорт",
		107: "Сотрудникик",
	}
	Directions = map[string]string{
		"ПОДШИПНИК":   "08d8a082-0b4c-4399-a72e-7675993b0519",
		"МОТО-СКУТЕР": "5acde960-a08a-4e45-9a95-d38e1c8bf4e6",
		"МЕТАН":       "7e535107-bfa5-4c90-8efe-484b9d5cba54",
		"ЛОГИСТИКА":   "1cda70ee-33af-4264-8641-485a471eb7b2",
		"ВЕЛО":        "9e7c8872-b869-4410-b70e-c09028491116",
		"АВТОШИНА":    "bd2b5583-43fa-4f23-b05e-6b7e4c8a33a3",
	}
	Currency = map[string]string{
		"USD": "b69448a1-c24c-4440-8d97-8d50e4204d4b",
		"UZS": "b95da2b5-6490-4eb4-81ac-64b13bc252bb",
	}
)
