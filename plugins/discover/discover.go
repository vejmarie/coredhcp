// Copyright 2024 Hewlett Packard Development LP.

// This package implement auto discover of GXP based HPE server

package discover

import (
	"bytes"
	"database/sql"
	"encoding/json"
        "archive/tar"
        "compress/gzip"
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
	"net"
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
	label	 string
}

type Roms struct {
    Roms []Rom `json:"roms"`
}

// User struct which contains a name
// a type and a list of social links
type Rom struct {
    Name   string `json:"name"`
    Systemid   []string `json:"systemid"`
    Id   []string `json:"id"`
}

var roms Roms

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
	Label      string
}

type Firmware struct {
        Version    string
        Date       string
}

func serveRPM(w http.ResponseWriter, r *http.Request) {
	b, _ := ioutil.ReadFile("/var/rpms/"+r.URL.Path)
	w.Write(b)
}

func ExtractTarGz(gzipStream io.Reader) string {
    uncompressedStream, err := gzip.NewReader(gzipStream)
    if err != nil {
        log.Fatal("ExtractTarGz: NewReader failed")
        log.Error("ExtractTarGz: NewReader failed")
        return "Error"
    }

    tarReader := tar.NewReader(uncompressedStream)

    for true {
        header, err := tarReader.Next()

        if err == io.EOF {
            break
        }

        if err != nil {
            log.Fatalf("ExtractTarGz: Next() failed: %s", err.Error())
        }

        switch header.Typeflag {
        case tar.TypeReg:
            outFile, err := os.Create(iscsiDir + "/target/tmp/"+ header.Name)
            if err != nil {
                log.Fatalf("ExtractTarGz: Create() failed: %s", err.Error())
            }
            if _, err := io.Copy(outFile, tarReader); err != nil {
                log.Fatalf("ExtractTarGz: Copy() failed: %s", err.Error())
            }
            outFile.Close()

        default:
            log.Fatalf(
                "ExtractTarGz: uknown type: %s in %s",
                header.Typeflag,
                header.Name)
        }
    }
    return "ok"
}

func serveROM(w http.ResponseWriter, r *http.Request) {
        // We must decode the system ID from the URI and deliver
        // the ROM file associated to it (latest version for the moment)
        systemID := string(r.URL.Path[1:])
        fmt.Println("ROM Request: "+systemID)
        for i := 0; i < len(roms.Roms); i++ {
                for j := 0; j < len(roms.Roms[i].Id); j++ {
                        if ( roms.Roms[i].Id[j] == systemID ) {
                                // Ok we need to serve the ROM
                                // Latest version is preferred
                                files, err := ioutil.ReadDir("/var/lib/iscsi_disks/roms/repo/images/")
                                if err != nil {
                                        log.Fatal(err)
                                }
                                mostRecent := time.Date(1900, time.Month(1), 1, 0, 0, 0, 0, time.UTC)
                                var romFile string
                                for _, file := range files {
                                        if ! file.IsDir() {
                                                if ( file.Name()[0:3] == roms.Roms[i].Name ) {
                                                        // We need to decode the date of that file
                                                        // only mtd files are of interest
                                                        if ( file.Name()[len(file.Name())-4:] == ".mtd" ) {
                                                                // Let's extract the date
                                                                fileDate := strings.Split(file.Name()[ len(file.Name()) - len("01_06_2023") -4 : len(file.Name())-4], "_")
                                                                year,_ := strconv.Atoi(fileDate[2])
                                                                month,_ := strconv.Atoi(fileDate[0])
                                                                day,_ := strconv.Atoi(fileDate[1])
                                                                firstTime := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
                                                                if ( firstTime.After(mostRecent) ) {
                                                                        mostRecent = firstTime
                                                                        romFile = file.Name()
                                                                }
                                                        }
                                                }
                                        }
                                }
                                if ( len(romFile) > 1 ) {
                                   b, _ := ioutil.ReadFile("/var/lib/iscsi_disks/roms/repo/images/"+romFile)
                                   w.Write([]byte(b))
                                }
                        }
                }
        }
}

func home(w http.ResponseWriter, r *http.Request) {
	head, tail := ShiftPath(r.URL.Path)
	var clients []Client
	var firmwares []Firmware
	switch head {
	case "js":
		w.Header().Add("Content-Type", "text/javascript")
		b, _ := ioutil.ReadFile(head + "/" + tail) // just pass the file name
		w.Write(b)
	case "css":
		w.Header().Add("Content-Type", "text/css")
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
        case "label":
                // The operation shall contain the client MAC and the new label
                type test_struct struct {
                    Mac string
                    Label string
                }
                decoder := json.NewDecoder(r.Body)
                var t test_struct
                err := decoder.Decode(&t)
                if err != nil {
                        log.Println("Error", err)
                }
                // So now we can update the label into the database
                p.ServersMac[t.Mac].label = t.Label
		newMac,_ := net.ParseMAC(t.Mac)
                p.saveServer(newMac, p.ServersMac[t.Mac])
                fmt.Fprintf(w,"")
        case "firmwares":
                var firmware Firmware
                files, err := ioutil.ReadDir(iscsiDir+"/images/")
                if err != nil {
                        log.Error(iscsiDir+"/images/ doesn't seem to exist ", err)
                        fmt.Fprintf(w,"{}")
                        return
                }
                for _, file := range files {
                        if file.IsDir() {
                                firmware.Version = file.Name()
                                firmware.Date = file.ModTime().String()
                                firmwares = append(firmwares, firmware)
                        }
                }
                b, _ := json.Marshal(firmwares)
                if len(b) == 0 {
                        b = []byte("{}")
                }

                fmt.Fprintf(w, "%s\n", b)
        case "upload_firmware":
                defer r.Body.Close()
                r.Body = http.MaxBytesReader(w, r.Body, 640<<20+4096)
                err := r.ParseMultipartForm(640<<20 + 4096)
                if err != nil {
                        fmt.Printf("Error %s\n", err.Error())
                }
                file, handler, err := r.FormFile("fichier")
                r.Body.Close()
                fmt.Printf("Uploading %s\n",handler.Filename);
                defer file.Close()
                f, err := os.OpenFile(iscsiDir + "/target/tmp/"+handler.Filename, os.O_WRONLY|os.O_CREATE, 0644)
                if err != nil {
                        fmt.Println(err)
                        return
                }
                defer f.Close()
                io.Copy(f, file)
                // We must validate the file and untar it first
                // It is gzipped and tar
                // Must find the version and create the associated directory into the target dir
                r, err := os.Open(iscsiDir + "/target/tmp/"+handler.Filename)
                if err != nil {
                        fmt.Println("error")
                }
                processTarball := ExtractTarGz(r)
                os.Remove(iscsiDir + "/target/tmp/"+handler.Filename)
                // We must read the VERSION file
                version, err := os.Open(iscsiDir + "/target/tmp/VERSION")
                if err != nil {
                                fmt.Println(err)
                }
                defer version.Close()
                versionValue, _ := ioutil.ReadAll(version)
                versionFinale := strings.TrimSpace(string(versionValue))
                os.Mkdir(iscsiDir+"/images/"+versionFinale, 0700)
                // push the boot.mtd and iscsi.tgt file into the right directory
                err = os.Rename(iscsiDir + "/target/tmp/boot.mtd" , iscsiDir+"/images/"+versionFinale+"/boot.mtd")
                err = os.Rename(iscsiDir + "/target/tmp/iscsi.tgt" , iscsiDir+"/images/"+versionFinale+"/iscsi.tgt")
                fmt.Fprintf(w,processTarball)
        case "images":
                w.Header().Add("Content-Type", "image/png")
                b, _ := ioutil.ReadFile(head + "/" + tail) // just pass the file name
                w.Write(b)
	case "clients":
		var counter int
		var client Client
		counter = 1

		for k, v := range p.ServersMac {
			client.MacAddress = k
			client.State = v.state
			client.IP = v.ip
			client.Label = v.label
			clients = append(clients, client)
			counter += 1
		}

		// let's get the number of targets
		// We shall get it from tgt daemon and not /etc/tgt/conf.d entries
		cmd := exec.Command("tgtadm", "--lld", "iscsi", "--op", "show", "--mode", "target")
		cOutput, _ := cmd.Output()
		j := strings.LastIndex(string(cOutput), "Target")
		if j == -1 {
			log.Error("Error Target not found " + string(cOutput))
			// No target found ... return empty strings
                        fmt.Fprintf(w,"{}")
                        return
		}
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
	// Do we have any ROM to serve ?
        files, err := ioutil.ReadDir("/var/lib/rom/configs/")
        if err != nil {
                log.Fatal(err)
        }

        for _, file := range files {
                fmt.Println(file.Name(), file.IsDir())
                if ! file.IsDir() {
                        jsonFile, err := os.Open("/var/lib/rom/configs/"+file.Name())
                        if err != nil {
                                fmt.Println(err)
                        }

                        fmt.Println("Successfully opened "+file.Name())
                        defer jsonFile.Close()
                        byteValue, _ := ioutil.ReadAll(jsonFile)

                        json.Unmarshal(byteValue, &roms)
                        for i := 0; i < len(roms.Roms); i++ {
                                for j := 0; j < len(roms.Roms[i].Id); j++ {
                                        fmt.Println("id ", roms.Roms[i].Name, roms.Roms[i].Id[j])
                                }
                        }
                }
        }


	// We start the webserver if needed
	if httpState {
		mux := http.NewServeMux()
		mux.HandleFunc("/", home)
		go http.ListenAndServe(":80", mux)
		mux8000 := http.NewServeMux()
		mux8000.HandleFunc("/", serveRPM)
		go http.ListenAndServe(":8000", mux8000)
                mux8080 := http.NewServeMux()
                mux8080.HandleFunc("/", serveROM)
                go http.ListenAndServe(":8080", mux8080)
	}

	return p.Handler4, nil
}

func (p *PluginState) Handler4(req, resp *dhcpv4.DHCPv4) (*dhcpv4.DHCPv4, bool) {
	var obfn dhcpv4.Option
	var clientOption []byte
	var localMac string
	rec := Record{
		state:    "new",
		bootfile: "boot.mtd",
		ip:       "",
		label:       "",
	}
	// we need to check if the server is within the database
	// if not we need to create the soft link to boot it to the default image
	// soft link is going to be boot_[mac address].mtd pointing to boot.mtd for the moment
	// If the server is in the database we don't do anything
        // client := resp.Options.OptionClientIdentifier()
	// clientOption = resp.Options.Get(dhcpv4.OptionClientIdentifier)
	clientOption = req.Options.Get(dhcpv4.OptionHostName)
	// let's extract the mac address
	if len(string(clientOption)) > 12 {
		localMac = string(clientOption[len(clientOption)-12:len(clientOption)-10])+
			":"+string(clientOption[len(clientOption)-10:len(clientOption)-8])+
			":"+string(clientOption[len(clientOption)-8:len(clientOption)-6])+
			":"+string(clientOption[len(clientOption)-6:len(clientOption)-4])+
			":"+string(clientOption[len(clientOption)-4:len(clientOption)-2])+
			":"+string(clientOption[len(clientOption)-2:]) 
	} else {
		localMac = ""
	}
        claddr, err := net.ParseMAC(localMac)
	if err == nil {
	        req.ClientHWAddr = claddr
	}
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
		f.WriteString("		vendor_id openbmc\n")
		f.WriteString("		product_id rootfs\n")
		f.WriteString("     </backing-store>\n")
	//	f.WriteString("     initiator-address " + resp.YourIPAddr.String() + "\n")
		f.WriteString("     initiator-name iqn.2016-04.com.open-iscsi:" + strings.ReplaceAll(req.ClientHWAddr.String(),":","") + "\n")
		f.WriteString("</target>\n")
		defer f.Close()
		rec.ip = resp.YourIPAddr.String()
		rec.label = "label";
		// We need to inform TGT daemon of the creation of the new target
		// this is done by issuing a tgt-admin --update ALL
		cmd := exec.Command("tgt-admin", "--update", "ALL")
		_, _ = cmd.Output()

		// We can create the server
		_ = p.saveServer(req.ClientHWAddr, &rec)
		// We need to save the record into the Map
		p.ServersMac[req.ClientHWAddr.String()] = &rec
		record, _ = p.ServersMac[req.ClientHWAddr.String()]
	} else {
		// update the IP
                record.ip = resp.YourIPAddr.String()
		p.ServersMac[req.ClientHWAddr.String()] = record
	}

	obfn = dhcpv4.OptBootFileName(record.bootfile)
	opt67 = &obfn
	resp.Options.Update(*opt67)
	return resp, false
}
