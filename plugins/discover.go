// Copyright 2024 Hewlett Packard Development LP.

// This package implement auto discover of GXP based HPE server

package discover

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/coredhcp/coredhcp/handler"
	"github.com/coredhcp/coredhcp/logger"
	"github.com/coredhcp/coredhcp/plugins"
	"github.com/insomniacslk/dhcp/dhcpv4"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	tftpDir      string
	iscsiDir     string
	tftpDefault  string
	iscsiDefault string
	httpState    bool
	err          error
	p            PluginState
)

var log = logger.GetLogger("plugins/discover")

// Plugin wraps plugin registration information
var Plugin = plugins.Plugin{
	Name:   "discover",
	Setup4: setup4,
}

var (
	opt67 *dhcpv4.Option
)

// Record holds an IP lease record
type Record struct {
	state    string
	bootfile string
	ip       string
}

// PluginState is the data held by an instance of the range plugin
type PluginState struct {
	// Rough lock for the whole plugin, we'll get better performance once we use leasestorage
	sync.Mutex
	// Recordsv4 holds a MAC -> state / bootfile mapping
	ServersMac map[string]*Record
	serverdb   *sql.DB
}

func parseArgs(args ...string) (*url.URL, error) {
	if len(args) != 4 {
		return nil, fmt.Errorf("Exactly four arguments must be passed to Discover plugin, got %d", len(args))
	}
	return url.Parse(args[0])
}

func ShiftPath(p string) (head, tail string) {
	p = path.Clean("/" + p)
	i := strings.Index(p[1:], "/") + 1
	if i <= 0 {
		return p[1:], "/"
	}
	return p[1:i], p[i:]
}

type Client struct {
	MacAddress string
	State      string
	IP         string
}

func home(w http.ResponseWriter, r *http.Request) {
	head, tail := ShiftPath(r.URL.Path)
	var clients []Client
	switch head {
	case "js":
		b, _ := ioutil.ReadFile(head + "/" + tail) // just pass the file name
		w.Write(b)
	case "css":
		b, _ := ioutil.ReadFile(head + "/" + tail) // just pass the file name
		w.Write(b)
	case "client":
		// We must delete the tftp image
		addr := tail[1:]
		addr = strings.Replace(addr, ":", "", -1)
		bootfile := "boot_" + addr + ".mtd"
		os.Remove(tftpDir + "/" + bootfile)
		// We must delete the TGT config and reload TGT
		os.Remove(iscsiDir + "/" + addr + "/disk01.img")
		os.Remove(iscsiDir + "/" + addr)
		// Now we need to update the TGT configuration
		os.Remove("/etc/tgt/conf.d/target" + addr + ".conf")
		cmd := exec.Command("tgt-admin", "--update", "ALL")
		_, _ = cmd.Output()
		// We have to remove the client from the list
		delete(p.ServersMac, tail[1:])
		_ = p.deleteServer(string(tail[1:]))
		// We need to wait a few to give iSCSI target monitoring timeout
		// enough time to declare the target released from initiator
		// 3s shall be good enough
		time.Sleep(3 * time.Second)
	case "clients":
		var counter int
		var client Client
		counter = 1

		for k, v := range p.ServersMac {
			client.MacAddress = k
			client.State = v.state
			client.IP = v.ip
			clients = append(clients, client)
			counter += 1
		}

		// let's get the number of targets
		// We shall get it from tgt daemon and not /etc/tgt/conf.d entries
		cmd := exec.Command("tgtadm", "--lld", "iscsi", "--op", "show", "--mode", "target")
		cOutput, _ := cmd.Output()
		j := strings.LastIndex(string(cOutput), "Target")
		k := strings.Index(string(cOutput[j:]), ":")
		myString := cOutput[j : j+k]
		biggestTargetNumber := myString[len("Target "):]
		entries, _ := strconv.Atoi(string(biggestTargetNumber))

		if err != nil {
			log.Fatal(err)
		}
		// let's report the target status
		for i := 1; i < (entries + 1); i++ {
			cmd := exec.Command("tgtadm", "--lld", "iscsi", "--op", "show", "--mode", "conn", "--tid", strconv.Itoa(i))
			cOutput, _ := cmd.Output()
			j := strings.Index(string(cOutput), "IP Address:")
			if j > 0 {
				IP := strings.TrimSuffix(string(cOutput[j+len("IP Address: "):]), "\n")
				for k, v := range clients {
					if v.IP == IP {
						clients[k].State = "Connected"
					}
				}
			}
		}
		b, _ := json.Marshal(clients)
		fmt.Fprintf(w, "%s\n", b)
	default:
		b, _ := ioutil.ReadFile("html/home.html")
		t := template.New("my template")
		buf := &bytes.Buffer{}
		t.Parse(string(b))
		t.Execute(buf, r.Host+"/")
		fmt.Fprintf(w, buf.String())
	}
}

func setup4(args ...string) (handler.Handler4, error) {
	if len(args) < 4 {
		return nil, fmt.Errorf("invalid number of arguments, want: 4 (tftp directory, initial image, iscsi target images directory, default iscsi image), got: %d", len(args))
	}

	var obfn dhcpv4.Option
	httpState = false
	if len(args) == 5 {
		if args[4] == "http" {
			httpState = true
		}
	}
	// That option is sending a default bootfile name
	// to the client
	tftpDefault = args[1]
	iscsiDefault = args[3]
	tftpDir = args[0]
	iscsiDir = args[2]
	if _, err := os.Stat(tftpDir); !os.IsNotExist(err) {
		if _, err := os.Stat(tftpDir + "/" + tftpDefault); !os.IsNotExist(err) {
			if _, err := os.Stat(iscsiDir); !os.IsNotExist(err) {
				if _, err := os.Stat(iscsiDir + "/" + iscsiDefault); !os.IsNotExist(err) {
				} else {
					return nil, fmt.Errorf("Error iscsi default target not found", err)
				}
			} else {
				return nil, fmt.Errorf("Error iscsi default dir doesn't exist", err)
			}
		} else {
			return nil, fmt.Errorf("Error tftp defaut file not found", err)
		}
	} else {
		return nil, fmt.Errorf("Error tftp default directory not found", err)
	}

	obfn = dhcpv4.OptBootFileName(args[1])

	opt67 = &obfn

	if err := p.registerBackingDB("servers.db"); err != nil {
		return nil, fmt.Errorf("could not setup lease storage: %w", err)
	}

	p.ServersMac, err = loadRecords(p.serverdb)
	if err != nil {
		return nil, fmt.Errorf("could not load records from file: %v", err)
	}

	// We start the webserver if needed
	if httpState {
		mux := http.NewServeMux()
		mux.HandleFunc("/", home)
		go http.ListenAndServe(":80", mux)
	}

	return p.Handler4, nil
}

func (p *PluginState) Handler4(req, resp *dhcpv4.DHCPv4) (*dhcpv4.DHCPv4, bool) {
	var obfn dhcpv4.Option
	rec := Record{
		state:    "new",
		bootfile: "boot.mtd",
		ip:       "",
	}
	// we need to check if the server is within the database
	// if not we need to create the soft link to boot it to the default image
	// soft link is going to be boot_[mac address].mtd pointing to boot.mtd for the moment
	// If the server is in the database we don't do anything
	record, ok := p.ServersMac[req.ClientHWAddr.String()]
	if !ok {
		// Server doesn't exist let's create it
		addr := req.ClientHWAddr.String()
		addr = strings.Replace(addr, ":", "", -1)
		rec.bootfile = "boot_" + addr + ".mtd"
		os.Symlink(tftpDir+"/"+tftpDefault, tftpDir+"/"+rec.bootfile)
		// We need to associate an iscsi target too
		os.Mkdir(iscsiDir+"/"+addr, 0700)
		// We need to copy the initial iscsi target file
		r, _ := os.Open(iscsiDir + "/" + iscsiDefault)
		defer r.Close() // ignore error: file was opened read-only.
		w, _ := os.Create(iscsiDir + "/" + addr + "/disk01.img")
		_, _ = io.Copy(w, r)
		// Now we need to update the TGT configuration
		f, _ := os.Create("/etc/tgt/conf.d/target" + addr + ".conf")
		f.WriteString("<target iqn.2022-04.world.srv:dlp.target" + addr + ">\n")
		f.WriteString("     # provided device as a iSCSI target\n")
		f.WriteString("     nop_count 1\n")
		f.WriteString("     nop_interval 2\n")
		f.WriteString("     <backing-store " + iscsiDir + "/" + addr + "/disk01.img>\n")
		f.WriteString("		lun 1\n")
		f.WriteString("		block-size 512\n")
		f.WriteString("		allow-in-use yes\n")
		f.WriteString("		write-cache on\n")
		f.WriteString("     </backing-store>\n")
		f.WriteString("     initiator-address " + resp.YourIPAddr.String() + "\n")
		f.WriteString("</target>\n")
		defer f.Close()
		rec.ip = resp.YourIPAddr.String()
		// We need to inform TGT daemon of the creation of the new target
		// this is done by issuing a tgt-admin --update ALL
		cmd := exec.Command("tgt-admin", "--update", "ALL")
		_, _ = cmd.Output()

		// We can create the server
		_ = p.saveServer(req.ClientHWAddr, &rec)
		// We need to save the record into the Map
		p.ServersMac[req.ClientHWAddr.String()] = &rec
		record, _ = p.ServersMac[req.ClientHWAddr.String()]
	}
	obfn = dhcpv4.OptBootFileName(record.bootfile)
	opt67 = &obfn
	resp.Options.Update(*opt67)
	return resp, true
}
