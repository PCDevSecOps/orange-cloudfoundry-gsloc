package healthchecks

import (
	"fmt"
	"github.com/gorilla/mux"
	hcconf "github.com/orange-cloudfoundry/gsloc-go-sdk/gsloc/api/config/healthchecks/v1"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/encoding/protojson"
	"io"
	"net/http"
	"sync"
)

type HcHandler struct {
	disabledEntIp *sync.Map
}

func NewHcHandler() *HcHandler {
	return &HcHandler{
		disabledEntIp: &sync.Map{},
	}
}

func (h *HcHandler) DisableEntryIp(fqdn, ip string) {
	log.Tracef(fmt.Sprintf("Disabling %s-%s", fqdn, ip))
	h.disabledEntIp.Store(fmt.Sprintf("%s-%s", fqdn, ip), struct{}{})
}

func (h *HcHandler) EnableEntryIp(fqdn, ip string) {
	log.Tracef(fmt.Sprintf("Enabling %s-%s", fqdn, ip))
	h.disabledEntIp.Delete(fmt.Sprintf("%s-%s", fqdn, ip))
}

func (h *HcHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	b, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	vars := mux.Vars(req)
	fqdn := vars["fqdn"]
	if fqdn == "" {
		http.Error(w, "fqdn is empty", http.StatusBadRequest)
		return
	}
	ip := vars["ip"]
	if ip == "" {
		http.Error(w, "ip is empty", http.StatusBadRequest)
		return
	}

	_, isDisabled := h.disabledEntIp.Load(fmt.Sprintf("%s-%s", fqdn, ip))
	if isDisabled {
		http.Error(w, "disabled entry", http.StatusGone)
		return
	}

	hcDef := &hcconf.HealthCheck{}
	err = protojson.Unmarshal(b, hcDef)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	hcker := MakeHealthCheck(hcDef, fqdn)
	host := fmt.Sprintf("%s:%d", ip, hcDef.GetPort())
	err = hcker.Check(host)
	if err != nil {
		http.Error(w, err.Error(), http.StatusExpectationFailed)
		return
	}
	w.WriteHeader(http.StatusOK)
}
