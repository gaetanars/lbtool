package livebox

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func postRequest(payload string) []byte {

	payloads := strings.NewReader(payload)

	req, _ := http.NewRequest("POST", URL, payloads)

	req.Header.Add("cookie", COOKIE)
	req.Header.Add("Content-Type", "application/x-sah-ws-4-call+json")
	req.Header.Add("X-Context", CONTEXTID)

	res, _ := http.DefaultClient.Do(req)

	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	if body == nil {
		log.Fatal("Error while executing request to Livebox")
	}

	return body
}

func displayFunboxValues(ip string) {
	url := "http://" + ip + "/sysbus/NeMo/Intf/data:getMIBs"

	req, _ := http.NewRequest("POST", url, nil)

	req.Header.Add("cookie", COOKIE)
	req.Header.Add("Accept", "text/javascript")
	req.Header.Add("X-Requested-With", "XMLHttpRequest")
	req.Header.Add("Content-type", "application/x-sah-ws-1-call+json")
	req.Header.Add("X-Context", CONTEXTID)

	res, _ := http.DefaultClient.Do(req)

	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	var fbvalues FunboxValues
	if err := json.Unmarshal(body, &fbvalues); err != nil {
		fmt.Println("Can not unmarshal JSON")
		os.Exit(1)
	} else {
		fmt.Println("===========UDM PRO SE SETTINGS===========")
		fmt.Println("PPPoE Username : " + fbvalues.Result.Status.Ppp.PppData.Username)
		fmt.Println("PPPoE Password : https://www.orange.pl/moj-orange -> Ustawienia -> Zabezpieczenia -> Hasło do Neostrady -> Zmień hasło")
		fmt.Println("VLAN ID		: " + strconv.Itoa(fbvalues.Result.Status.Vlan.GvlanData.Vlanid))
		fmt.Println("=========================================")

		GponSn = fbvalues.Result.Status.Gpon.Veip0.SerialNumber
		PonVendorId = fbvalues.Result.Status.Gpon.Veip0.SerialNumber[0:4]

		HwHwver = fbvalues.Result.Status.Gpon.Veip0.HardwareVersion
		OmciSwVer1 = fbvalues.Result.Status.Gpon.Veip0.ONTSoftwareVersion0
		OmciSwVer2 = fbvalues.Result.Status.Gpon.Veip0.ONTSoftwareVersion1

		fmt.Println("")
		fmt.Println("============LEOX GPON COMMAND============")
		generateGponCommands()
		fmt.Println("=========================================")
	}
}

func getOntInfos() {
	body := postRequest("{\"service\":\"NeMo.Intf.veip0\",\"method\":\"getMIBs\",\"parameters\":{\"mibs\":\"gpon\"}}")

	var ont Ont

	if err := json.Unmarshal(body, &ont); err != nil {
		fmt.Println("Can not unmarshal JSON")
		os.Exit(1)
	} else {
		GponSn = ont.Status.Gpon.Veip0.SerialNumber
		PonVendorId = ont.Status.Gpon.Veip0.VendorID
		HwHwver = ont.Status.Gpon.Veip0.HardwareVersion
		OmciSwVer1 = ont.Status.Gpon.Veip0.ONTSoftwareVersion0
		OmciSwVer2 = ont.Status.Gpon.Veip0.ONTSoftwareVersion1
	}
}

// Fetch Mac Address
func getMacAddress() {
	body := postRequest("{\"service\":\"NMC\",\"method\":\"getWANStatus\",\"parameters\":{}}")

	var mac MacAddress

	if err := json.Unmarshal(body, &mac); err != nil {
		fmt.Println("Can not unmarshal JSON")
		os.Exit(1)
	} else {
		macaddress = mac.Data.MACAddress
	}
}

// Fetch Internet VLAN
func getInternetVlan() {
	body := postRequest("{\"service\":\"NeMo.Intf.data\",\"method\":\"getFirstParameter\",\"parameters\":{\"name\":\"VLANID\"}}")

	var vlaninternet VlanInternet

	if err := json.Unmarshal(body, &vlaninternet); err != nil {
		fmt.Println("Can not unmarshal JSON")
		os.Exit(1)
	} else {
		vlanid = strconv.Itoa(vlaninternet.Status)
	}
}

// Fetch DHCP Infos
func GetDHCPInfos() {
	body := postRequest("{\"service\":\"NeMo.Intf.data\",\"method\":\"getMIBs\",\"parameters\":{\"mibs\":\"dhcp\"}}")

	var dhcpinfos DHCP

	if err := json.Unmarshal(body, &dhcpinfos); err != nil {
		fmt.Println("Can not unmarshal JSON")
		os.Exit(1)
	}

	option70 := dhcpinfos.Status.Dhcp.DhcpData.SentOption.Num77.Value
	option70decoded, err := hex.DecodeString(option70)
	if err != nil {
		fmt.Println("Erreur")
	}

	option70decodedstring := string(option70decoded)

	opt70 := strings.Split(option70decodedstring, "+")

	dhcpoption77 = opt70[1]

	// Add : every 2 char for DHCP Option 90
	var buffer bytes.Buffer
	var n1 = 2 - 1
	var l1 = len(dhcpinfos.Status.Dhcp.DhcpData.SentOption.Num90.Value) - 1
	for i, runner := range dhcpinfos.Status.Dhcp.DhcpData.SentOption.Num90.Value {
		buffer.WriteRune(runner)
		if i%2 == n1 && i != l1 {
			buffer.WriteRune(':')
		}
	}
	dhcpoption90 = buffer.String()
}

func generateOMCC(oltvendorid string) {
	id := "128"
	dict := map[string]string{"HWTC": "136", "ALCL": "128"}

	if value, ok := dict[oltvendorid]; ok {
		id = value
	} else {
		fmt.Println("Unknown OLT VENDOR ID, defaulting to 128")
	}

	fmt.Println("\nExecute this command -> flash set OMCC_VER " + id)

	if oltvendorid == "ALCL" {
		if len(HwHwver) != 0 {
			fmt.Println("flash set HW_HWVER " + HwHwver)
		}
		fmt.Println("flash set OMCI_SW_VER1 " + OmciSwVer1)
		fmt.Println("flash set OMCI_SW_VER2 " + OmciSwVer2)
		fmt.Println("flash set OMCI_TM_OPT 0")
		fmt.Println("flash set OMCI_OLT_MODE 1")
	}
}

func generateGponCommands() {
	var oltvendorid string

	//fmt.Println("flash set GPON_PLOAM_PASSWD DEFAULT012")
	fmt.Println("flash set GPON_SN " + GponSn)
	fmt.Println("flash set PON_VENDOR_ID " + PonVendorId)
	fmt.Println("\n## Unplug fiber from Livebox and plug it into UDM and wait a minute ##\n")
	fmt.Println("Execute this command -> omcicli mib get 131")

	fmt.Print("\nOLT VENDOR ID (HWTC, ALCL, ...) : ")
	fmt.Scan(&oltvendorid)
	generateOMCC(strings.ToUpper(oltvendorid))
}

func displayUDMinfos() {
	fmt.Println("NAME              : LEOX GPON")
	fmt.Println("VLAN ID           : " + vlanid)
	fmt.Println("MAC Address Clone : " + macaddress)
	fmt.Println("DHCP OPTION 60    : sagem")
	fmt.Println("DHCP OPTION 77    : " + dhcpoption77)
	fmt.Println("DHCP OPTION 90    : " + dhcpoption90)
	fmt.Println("DHCP CoS          : 6 (Requires Network App 7.4.X or later)")
}
