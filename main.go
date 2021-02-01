package main

import (
	"bytes"
	"encoding/json"
        "encoding/csv"
	"fmt"
        "time"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"net/http"
	"os"
        "golang.org/x/crypto/ssh"
        "golang.org/x/crypto/ssh/terminal"
	"sync"
 	"strings"
	"strconv"
	"net"
	"net/url"
	"flag"
        "syscall"
	"runtime"
	"runtime/debug"
	"github.com/Juniper/go-netconf/netconf"
	"regexp"
	log "github.com/golang/glog"
	influxdb_client "github.com/influxdata/influxdb1-client"
)

var const_time time.Time
const (
    layoutISO = "2006-01-02"
)
var global_orch *Orchestrator 

var (
        vmx_device         = flag.String("host", "10.40.55.112:830", "vmx1")
        username     = flag.String("username", "gxtech", "Username")
        key          = flag.String("key", os.Getenv("HOME")+"/.ssh/id_rsa", "SSH private key file")
        passphrase   = flag.String("passphrase", "gxtech", "SSH private key passphrase (cleartext)")
        nopassphrase = flag.Bool("nopassphrase", false, "SSH private key does not contain a passphrase")
        pubkey       = flag.Bool("pubkey", false, "Use SSH public key authentication")
        agent        = flag.Bool("agent", false, "Use SSH agent for public key authentication")
)


func isError(err error) bool {
    if err != nil {
        fmt.Println(err.Error())
    }

    return (err != nil)
}


func checkErr(err error) {
	if err != nil {
		fmt.Printf("error: %+v\n", err)
		panic(err)
	}
}

// RawMethod defines how a raw text request will be responded to
type RawMethod string

type SVNData struct {
        SVN uint `json:"SVN"`
        SAT0IP string `json:"SAT0IP"`
        SAT0SUBNET string `json:"SAT0SUBNET"`
        ETH0IP  string `json:"ETH0IP"`
        ETH0SUBNET string `json:"ETH0IP"`
}

type DALDataStructure struct {
        Time uint64  `json:"TIMEMS"`
	DID uint64   `json:"DID"`
	LAT  float32 `json:"LAT"`
        LON  float32 `json:"LON"`
        SAC  uint  `json:"SAC"`
        SVNs []SVNData `json:"SVNS"`
}

type InfluxDBDALData struct {
	Time uint64
	DID  string
	SAC  string
	SVNs string
}



type DeviceInfo struct {
        Name string `json:"name"`
	Ipaddress string `json:"ipaddress"`
	Username string `json:"username"`
        Password string `json:"password"`
}

type SVNInfo struct {
        Name uint `json:"name"`
        Operation string `json:"operation"`
}

type DeviceConfig struct {
        Devicelist []DeviceInfo `json:"devicelist"`
}

type Config struct {
	SDNc    string    `json:"sdnc"`
	Subnets_file string `json:"subnets_file"`
	Kafka_server_list string `json:kafka_server_list"`
	India_sac_list string `json:"india_sac_list"`
	Devicelist []DeviceInfo `json:"devicelist"`
        SVNlist []SVNInfo `json:"svnlist"`
}

type Orchestrator struct {
	config             Config
	debug_enabled      bool
	remotelist         map[uint32]OrchRemote
        dALTerminalData    map[uint64]DALDataStructure
	Terminal_IPs_List  []Terminal_IPs
	DalKafkaMessage	   chan	string 
	mx_subnets    	   map[string]uint64
	SubnetsKafkaMessage    chan string
	SubnetsCSVLine     chan []string 
	influxdb           *influxdb_client.Client //*http.Client    

}

type Terminal_IPs struct {
  Terminal_did  uint64    `csv:"terminal_did"`
  //Terminal_name string `csv:"terminal_name"`
  SVN           uint `csv:"svn"`
  Subnet        string `csv:"subnet"`
  mx_subnets    [] string
}

func str2uint(str string) (uint, error) {
	u64, err := strconv.ParseUint(str, 10, 32)
	if err != nil {
        	return 0, err
        }
	return  uint(u64),err
}


func str2uint64(str string) (uint64, error) {
	i, err := strconv.ParseInt(str, 10, 64)
	return uint64(i), err
}

func uint2string (val uint) (string) {
	
	str := fmt.Sprint(val)
	return str
}

func uint642string (val uint64) (string) {

    // Format to a string by passing the number and it's base.
    str := strconv.FormatUint(val, 10)
    return str
}

func bToMb(b uint64) string {
    return uint642string(b / 1024 / 1024)
}

func findIP(input string) []string {
         numBlock := "(25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])"
         regexPattern := numBlock + "\\." + numBlock + "\\." + numBlock + "\\." + numBlock

         regEx := regexp.MustCompile(regexPattern)
         return regEx.FindAllString(input, -1)
}

func (orch *Orchestrator) printConfiguration () {
	fmt.Printf("India sac code list %s\n", orch.config.India_sac_list)
	fmt.Printf("Subets located at %s\n", orch.config.Subnets_file)
        fmt.Printf("Kafka_server_list %s\n", orch.config.Kafka_server_list)
        for _, deviceinfo := range orch.config.Devicelist {
        	fmt.Printf("Device  name %s\n", deviceinfo.Name)
                fmt.Printf("Device  ipaddress %s\n", deviceinfo.Ipaddress)
                fmt.Printf("Device  username %s\n", deviceinfo.Username)
                fmt.Printf("Device  password %s\n", deviceinfo.Password)
        }
        for _, svninfo := range orch.config.SVNlist {
        	fmt.Printf("SVN name %d\n", svninfo.Name)
                fmt.Printf("SVN  operation %s\n", svninfo.Operation)
        }
}

func (orch *Orchestrator) Configure() {

	//fmt.Printf("Configure from config-sdn-orch.json \n")
	configfile, err := os.Open("config-sdn-orch.json")
	checkErr(err)
	jsondec := json.NewDecoder(configfile)
	err = jsondec.Decode(&orch.config)
	checkErr(err)
	if (orch.debug_enabled) {
        	fmt.Printf("India sac code list %s\n", orch.config.India_sac_list)
        	fmt.Printf("Kafka_server_list %s\n", orch.config.Kafka_server_list)
        	for _, deviceinfo := range orch.config.Devicelist {
                	fmt.Printf("Device  name %s\n", deviceinfo.Name)
			fmt.Printf("Device  ipaddress %s\n", deviceinfo.Ipaddress)
 			fmt.Printf("Device  username %s\n", deviceinfo.Username)
                	fmt.Printf("Device  password %s\n", deviceinfo.Password)
        	}
		for _, svninfo := range orch.config.SVNlist {
                	fmt.Printf("SVN name %d\n", svninfo.Name)
                	fmt.Printf("SVN  operation %s\n", svninfo.Operation)
        	}
	}


	checkErr(err)
	//orch.consul = client
	orch.debug_enabled = false
	orch.remotelist = make(map[uint32]OrchRemote)
        orch.dALTerminalData =  make(map[uint64]DALDataStructure)
	orch.DalKafkaMessage = make(chan string, 10) 
	orch.SubnetsKafkaMessage = make(chan string, 10)
	orch.SubnetsCSVLine = make (chan []string, 10)
	host, err := url.Parse(fmt.Sprintf("http://%s:%d", "localhost", 8086))
	if err != nil {
		log.Fatal(err)
	}
	conf := influxdb_client.Config{
		URL:      *host,
	}
	con, err := influxdb_client.NewClient(conf)
	if err != nil {
		log.Fatal(err)
	}
	orch.influxdb = con
}

func (orch *Orchestrator) DALConsumerStart (wg *sync.WaitGroup, dal_url string) {

        var dalKafkainfo string // rchDALKafkaMessage
        dal, err := kafka.NewConsumer(&kafka.ConfigMap{
                "bootstrap.servers": dal_url,
                "group.id":          "myGroup",
                "auto.offset.reset": "earliest",
        })
	if (orch.debug_enabled) {
		fmt.Printf("SubscribeTopics dal_url %s \n", dal_url)
	}
	dal.SubscribeTopics([]string{"india-sdn", "^aRegex.*[Tt]opic"}, nil)

        if err != nil {
                panic(err)
        }
        for {
                msg, err := dal.ReadMessage(-1)
                if err == nil {
			dalKafkainfo = string(msg.Value)
                        orch.DalKafkaMessage <- dalKafkainfo
                } else {
                        // The client will automatically try to recover from all errors.
                        fmt.Printf("DAL Consumer error: %v (%v)\n", err, msg)
                }
        }
        dal.Close()
        wg.Done()
}

func (orch *Orchestrator) SubnetsConsumerStart (wg *sync.WaitGroup, subnet_url string) {

        var subnetsKafkainfo string // rchDALKafkaMessage
        subs, err := kafka.NewConsumer(&kafka.ConfigMap{
                "bootstrap.servers": subnet_url,
                "group.id":          "myGroup",
                "auto.offset.reset": "earliest",
        })
        if (orch.debug_enabled) {
                fmt.Printf("SubscribeTopics dal_url %s \n", subnet_url)
        }
        subs.SubscribeTopics([]string{"subnets_update", "^aRegex.*[Tt]opic"}, nil)

        if err != nil {
                panic(err)
        }
        for {
                msg, err := subs.ReadMessage(-1)
                if err == nil {
                        subnetsKafkainfo = string(msg.Value)
                        orch.SubnetsKafkaMessage <- subnetsKafkainfo 
                } else {
                        // The client will automatically try to recover from all errors.
                        fmt.Printf("Subnets Consumer error: %v (%v)\n", err, msg)
                }
        }
        subs.Close()
        wg.Done()
}


func StartKafkaConsumer(orch *Orchestrator, wg sync.WaitGroup) {

        fmt.Printf("StartKafkaConsumer ...\n")
        kafka_server_array := strings.Split(orch.config.Kafka_server_list, ",")
	for _, kafka_server_info := range kafka_server_array {
        	go orch.DALConsumerStart (&wg, kafka_server_info)
	}
}

func BuildConfig(uname string, password string) *ssh.ClientConfig {
        var config *ssh.ClientConfig
        var pass string
        if *pubkey {
                if *agent {
                        var err error
                        config, err = netconf.SSHConfigPubKeyAgent(uname)
                        if err != nil {
                                log.Fatal(err)
                        }
                } else {
                        if *nopassphrase {
                                pass = "\n"
                        } else {
                                if *passphrase != "" {
                                        pass = *passphrase
                                } else {
                                        var readpass []byte
                                        var err error
                                        fmt.Printf("Enter Passphrase for %s: ", *key)
                                        readpass, err = terminal.ReadPassword(syscall.Stdin)
                                        if err != nil {
                                                log.Fatal(err)
                                        }
                                        pass = string(readpass)
                                        fmt.Println()
                                }
                        }
                        var err error
                        config, err = netconf.SSHConfigPubKeyFile(uname, *key, pass)
                        if err != nil {
                                log.Fatal(err)
                        }
                }
        } else {
                config = netconf.SSHConfigPassword(uname, password)
        }
        return config
}

func ReadCsv(filename string) ([][]string, error) {
    f, err := os.Open(filename)
    if err != nil {
        return [][]string{}, err
    }
    defer f.Close()
    lines, err := csv.NewReader(f).ReadAll()
    if err != nil {
        log.Fatal(err)
    }
    return lines, nil
}

func LoadConfig(orch *Orchestrator) {
        configfile, err := os.Open("config-sdn-orch.json")
        checkErr(err)
        jsondec := json.NewDecoder(configfile)
        err = jsondec.Decode(&orch.config)
        checkErr(err)
        //if (orch.debug_enabled) {
                fmt.Printf("India sac code list %s\n", orch.config.India_sac_list)
                fmt.Printf("Kafka_server_list %s\n", orch.config.Kafka_server_list)
                for _, deviceinfo := range orch.config.Devicelist {
                        fmt.Printf("Device  name %s\n", deviceinfo.Name)
                        fmt.Printf("Device  ipaddress %s\n", deviceinfo.Ipaddress)
                        fmt.Printf("Device  username %s\n", deviceinfo.Username)
                        fmt.Printf("Device  password %s\n", deviceinfo.Password)
                }
                for _, svninfo := range orch.config.SVNlist {
                        fmt.Printf("SVN  name %d\n", svninfo.Name)
                        fmt.Printf("SVN  operation %s\n", svninfo.Operation)
                }
        //}
}

func LoadSubnets(orch *Orchestrator) {

    lines, err := ReadCsv(orch.config.Subnets_file) // "subnets.txt")
    if err != nil {
        fmt.Println("LoadSubnets point 3 \n")
        panic(err)
    }
    //var add_entry bool = true
    //var add_svn bool = false
    // var terminal_ips_array []Terminal_IPs
    orch.Terminal_IPs_List = nil
    // Loop through lines & turn into object
    for _, line := range lines {
	if (strings.Contains(line[0], "terminal_did")) {
		// ignore ther header
		continue
		
	}
        did, _ := strconv.ParseUint(line[0], 10, 64)
        //name := line[1]
        u64, err := strconv.ParseUint(line[1], 10, 32)
        if err != nil {
            fmt.Println("SAC_CONTROLLER SVN err %v", err)
            continue
        }
        svn := uint(u64)
        sub := line[2]
        data := Terminal_IPs{
                Terminal_did : did,
                //Terminal_name: name,
                SVN: svn,
                Subnet: sub,
        }
        //fmt.Println("data.Terminal_did %d \n", data.Terminal_did)
        //fmt.Println("data.Terminal_name %s \n", data.Terminal_name)
        //fmt.Println("data.SVN %d \n", data.SVN)
        //fmt.Println("data.Subnet %s \n", data.Subnet)
        // check for duplicate did and subnet before adding
        //for _, a := range orch.Terminal_IPs_List {
                // only care about svn's  in the config list
        for _, element := range orch.config.SVNlist {
        	if (element.Name == data.SVN) {
                                //add_svn = true
				orch.Terminal_IPs_List = append(orch.Terminal_IPs_List, data)
                                break
                }
        }

    }
    fmt.Println("subnets.csv loaded len \n", len(orch.Terminal_IPs_List))
}


func HTTPServer(w http.ResponseWriter, r *http.Request) {
    	
        fmt.Println("HTTPServer, %s!", r.URL.Path[1:])
        if ( r.URL.Path[1:] == "get-terminal-data") {
		fmt.Println("SAC Controller App currently holds information on this number of terminals : \n", len (global_orch.dALTerminalData))
		var terminal_data string =  "Terminal     SAC Subnets \n"
		for key, element := range global_orch.dALTerminalData {
			// get the associated subnets for this terminal
			var subnets_str string
			//var svn_list string
			for _, a := range global_orch.Terminal_IPs_List {
                		if (a.Terminal_did == uint64(key)) {
                        		subnets_str = subnets_str +  a.Subnet + " "
                        		
                		}

        		}
			terminal_data = terminal_data +  strconv.Itoa(int(key)) + "  " + "  " + strconv.Itoa(int(element.SAC)) + " " + subnets_str + "\n"
    		}
 		w.Write([]byte(terminal_data))
		terminal_data = ""
	} else if (r.URL.Path[1:] == "get-india-only-terminal-data") {

		var add_entry bool = false
                var terminal_data string =  "Terminal     SAC Subnets  \n"
		india_sac_array := strings.Split(global_orch.config.India_sac_list, ",")
		
                for key, element := range global_orch.dALTerminalData {

			for _, sac_code := range india_sac_array {
                        	if (strconv.Itoa(int(element.SAC)) == sac_code) {
                        	        add_entry = true
                	                break
        	                }
	                }
			if (add_entry) {
                        	// get the associated subnets for this terminal
                        	var subnets_str string
                        	for _, a := range global_orch.Terminal_IPs_List {
                                	if (a.Terminal_did == uint64(key)) {
                                        	subnets_str = subnets_str +  a.Subnet + " "

                                	}
                        	}
                        	terminal_data = terminal_data +  strconv.Itoa(int(key)) + "  " + "  " + strconv.Itoa(int(element.SAC)) + " " + subnets_str + "\n"
			}
			add_entry=false
                }
                w.Write([]byte(terminal_data))

        } else if (r.URL.Path[1:] == "set-terminal-data") {
		buf := new(bytes.Buffer)
    		buf.ReadFrom(r.Body)
    		json_data := buf.String()
		if (global_orch.debug_enabled) {
			fmt.Println("HTTPServer - set-termiinal-data", r.URL.Path)
        		fmt.Println("json_data %s\n",json_data)
		}
  		var dalKafkainfo string = json_data
                global_orch.DalKafkaMessage <- dalKafkainfo
	}  else if (r.URL.Path[1:] == "load-terminal-ips") {
		if (global_orch.debug_enabled) {
                	fmt.Println("HTTPServer - load_terminal_ips ", r.URL.Path)
		}
                LoadSubnets(global_orch) 

        } else if (r.URL.Path[1:] == "load-config") {
		if (global_orch.debug_enabled) {
                	fmt.Println("HTTPServer - load-config %s!", r.URL.Path)
		}
		LoadConfig(global_orch)
        } else if (r.URL.Path[1:] == "enable-debug") {
		if (global_orch.debug_enabled) {
                	fmt.Println("HTTPServer - enable-debug %s!", r.URL.Path)
		}
		global_orch.debug_enabled = true
        } else if (r.URL.Path[1:] == "disable-debug") {
		if (global_orch.debug_enabled) {
                	fmt.Println("HTTPServer - disable-debug %s!", r.URL.Path)
		}
		global_orch.debug_enabled = false
        } else if (r.URL.Path[1:] == "print-mem-stats") {
                var m runtime.MemStats
                runtime.ReadMemStats(&m)
                var mem_data_header string =  "Alloc   TotalAlloc   Sys NumGC   \n"
                alloc := bToMb(m.Alloc)
                t_alloc :=  bToMb(m.TotalAlloc)
                mem_sys := bToMb(m.Sys)
                num_gc :=  bToMb(uint64(m.NumGC))
                var mem_data string = alloc + "       " +  t_alloc + "            " + mem_sys + "  " + num_gc + "\n"
                mem_data = mem_data_header + mem_data
                w.Write([]byte(mem_data))
	
	} else if (r.URL.Path[1:] == "print-config") {
                global_orch.printConfiguration()

        } else if (r.URL.Path[1:] == "cause-crash") {
		os.Exit(mainReturnWithCode())
	} 
}

// Start on port  listenini8080 for http requests 
func startListeningForHTTP(orch *Orchestrator) {

	s := &http.Server{
		Addr:           ":8080",
		Handler:        http.HandlerFunc(HTTPServer),  
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	go func () { 
    		s.ListenAndServe() // ":8080", nil)
	}()
	
	global_orch = orch
	fmt.Println("SAC Controller... startListening for API calls on port 8080 ...\n")
	for {
		;
	}
}


func mainReturnWithCode() int {
    // do stuff, defer functions, etc.
    exitcode := 0
    return exitcode // a suitable exit code
}

func MethodGetDeviceLicensedFilterConfig(source string, svn_id string) RawMethod {
        // move this xml to a proper container file
	using_string := "<get-config><source><%s/></source><filter type=\"subtree\"><configuration><firewall><family><inet><filter><name>10mile</filter></inet></family></firewall></configuration></filter></get-config>"


	using_string = strings.Replace(using_string, "SVN-ID", svn_id,  -1)
	
	fmt.Println("MethodGetConfig using_string %s\n", using_string)
	return RawMethod(using_string)

}


func netconf_operation (Devicelist []DeviceInfo, terminal_subnets  map[int]string, svn string, set bool, licensed bool) bool {
	var op_success bool = true
	for _, device := range Devicelist {
		if (global_orch.debug_enabled) {
        		fmt.Printf("netconf_operation mx_ip_address %s set %t licensed %t \n",device.Ipaddress, set, licensed)
		}	
		config := BuildConfig(device.Username, device.Password)
        	s, err := netconf.DialSSH(device.Ipaddress, config)
        	if err != nil {
	  		fmt.Println("SAC_CONTROLLER NETCONF  err condition hit, command not written %v \n", err)
			continue
            		//log.Fatal(err)
        	}

        	defer s.Close()
		var ip_count int = 0
		var prefix_name string
		var prefix_list string // = "prefix-list-item>"
		var termSet string = GetXmlSetPrefixList()
		if (licensed) {
			 prefix_name = "SVN" + svn + "_Licensed"
		} else {
			prefix_name = "SVN" + svn + "_Not_Licensed"
		}
		termSet = strings.Replace(termSet, "prefix-name", prefix_name,  -1)
		
                for _,ip := range terminal_subnets {
			_, net_ip, _ := net.ParseCIDR(ip)
                	if (ip_count == len(terminal_subnets)-1) {
				if (set) {
                                	prefix_list = prefix_list + "<prefix-list-item><name>" + net_ip.String() + "</name></prefix-list-item"
				} else {
				        prefix_list = prefix_list + `<prefix-list-item operation="delete"><name>` + net_ip.String() + "</name></prefix-list-item"
				}

                        } else {
			        if (set) {
                               		prefix_list = prefix_list + "<prefix-list-item><name>" + net_ip.String() + "</name></prefix-list-item>"
				} else {
					prefix_list = prefix_list + `<prefix-list-item operation="delete"><name>` + net_ip.String() + "</name></prefix-list-item>"
				}
                        }
                        ip_count++
                }
		termSet =  strings.Replace(termSet, "<placeholder-prefix-list-item", prefix_list,  -1)
        	t := time.Now()
        	date_time := t.UTC().Format(http.TimeFormat)
        	termSet =  strings.Replace(termSet, "date-time", date_time,  -1)

		if (global_orch.debug_enabled) {
        		fmt.Println("netconf_operation Exec termSet \n", termSet)
		}
        	reply, err := s.Exec(netconf.RawMethod (termSet))
        	if err != nil {
	      		s.Close()
			op_success = false
              		fmt.Printf("netconf_operation returned error %v", err)
			continue
        	}
		if (global_orch.debug_enabled) {
        		fmt.Printf("Netconf returns  %v", reply)
		}
		
	}
 	if (global_orch.debug_enabled) {
                        fmt.Printf("netconf_operation returns %t \n",op_success)
        }
	for k := range terminal_subnets {
    		delete(terminal_subnets, k)
	}
	return (op_success)
}

func restoreDALData(orch *Orchestrator) {

	fmt.Printf("restoreDALData from Influx \n")
	q := influxdb_client.Query{
		Command:  "select * from DALData",
		Database: "DALData",
	}
	if response, err := orch.influxdb.Query(q); err == nil && response.Error() == nil {
		for  _, r := range response.Results {
			for _, s := range r.Series {
				for _, v := range s.Values {
					if (len (v) >= 5) {
						//fmt.Printf("restoreDALData v DID %v \n", v, v[1] ) 
						var dalData DALDataStructure
						did_str := fmt.Sprintf("%v", v[1])
						if num, err := str2uint64(did_str); err == nil {
    							dalData.DID = num
						}
						sac_str := fmt.Sprintf("%v", v[2])
						u64, err := strconv.ParseUint(sac_str, 10, 32)
    						if err == nil {
        						dalData.SAC = uint(u64)
    						}
						svn_str := fmt.Sprintf("%v", v[3])
						dalSVNs := strings.Split(svn_str, ",")
						if (len (dalSVNs) > 0) {
							dalData.SVNs = make([]SVNData, len (dalSVNs))
							for i, s := range dalSVNs {
								if num, err := str2uint(s); err == nil {
									//append(dalData.SVNs, data)
									dalData.SVNs[i].SVN = num
								}
							}
						}
						time_str := fmt.Sprintf("%v", v[4])
                                                if time_num, err := str2uint64(time_str); err == nil {
                                                        dalData.Time = time_num
                                                }

						orch.dALTerminalData[dalData.DID] = dalData
					}
				}
			}
		}
	} else {
		fmt.Printf("Query failed %v \n", err)
	}
}


func backupDALData(orch *Orchestrator, data DALDataStructure) {
	sac := data.SAC
	time := uint642string(data.Time)
        did := strconv.FormatUint(data.DID, 10)
        var svns string
        for _, svn := range data.SVNs {
                var n uint = svn.SVN
                svn_str := fmt.Sprint(n)
                svns = svns + svn_str + ","
        }

        bp := influxdb_client.BatchPoints{
        Database:  "DALData",
        Points: []influxdb_client.Point{
                {
                                Measurement: "DALData",
                                Tags: map[string]string{
                                        "terminal": did,
                                },
                                Time: const_time, // using a const time because I only want a single entry per DID in the database for its latest SAC code//time.Now(),
				Precision: "s",
                                Fields: map[string]interface{}{
                                        "DID":did,
                                        "SAC": sac,
                                        "SVNs": svns,
					"TIME": time,
                                },
                        },
                },
        }
	if (orch.debug_enabled) {
                fmt.Printf("Writing data to database for DID %s SAC %d and svns %s time %s\n",did, sac, svns, time)
	}
	r, err := orch.influxdb.Write(bp)
        if err != nil {
                fmt.Printf("influxdb write unexpected error.  expected %v, actual %v", nil, err)
        }
        if r != nil {
                fmt.Printf("influxdb unexpected response. expected %v, actual %v", nil, r)
        }
}



func startAutomation (orch *Orchestrator) {

        var wg sync.WaitGroup
	StartKafkaConsumer(orch, wg)
	fmt.Printf("startAutomation dalData len %d \n", len(orch.dALTerminalData))
	for {
                select {
                        case dal_message := <-orch.DalKafkaMessage:
				if (orch.debug_enabled) {
                        		fmt.Printf("DAL Message received (json formatted) %s\n",dal_message) //dal_message.DALmessage)
				}
				var dalData DALDataStructure
                		u_err := json.Unmarshal([]byte(dal_message), &dalData)
                		if u_err != nil {
                        		fmt.Println("sac controller failed to unmarshal DAL kafka data", u_err)
                		} else {
					if (orch.debug_enabled) {
  		                		fmt.Println("Dal message for DID %d with SAC %d \n",dalData.DID, dalData.SAC)
					}
				
					//fmt.Println("Message received at Time %d \n", dalData.LAT)
					//fmt.Println("Message received at Time %d \n", dalData.LON)
					//fmt.Println("Message received at Time %d \n", dalData.Time)
					// add to dAL list with new terminal or update existing details
					// if sac code changes bewteen in/out make netconf call
					india_sac_array := strings.Split(orch.config.India_sac_list, ",")
					current_terminal_data, found := orch.dALTerminalData[dalData.DID]
                			if found {
						if (orch.debug_enabled) {	
							fmt.Printf("DAL Message Terminal is found for DID %d \n", dalData.DID)
						}
						if (dalData.Time < current_terminal_data.Time) {
						        fmt.Printf("DAL Message Terminal is old for DID %d \n", dalData.DID)
							fmt.Printf("DAL Message current held time %d \n", current_terminal_data.Time)
							fmt.Printf("DAL Message dal update time (old) %d \n", dalData.Time)
							continue
						}
                        			// this terminal is already known to me. What does the change of SAC code mean ?
						// 4 scenarios.
						// currently in India sac -> moved to another India sac              = no action
						// currently in India sac -> moved outside India sac                 = take nectonf action (remove)
						// currently outside India sac -> moved to another outside India sac = no action
						// currently outside India sac -> moved inside India sac             = take netconf action (add)      
                        			if current_terminal_data.SAC !=  dalData.SAC {
							if (orch.debug_enabled) {
								fmt.Printf("existing sac %d new sac %d \n",  current_terminal_data.SAC, dalData.SAC)
							}
							// sac code has changed. has it moved into or out of an India area ?
							mx_operation_applied := false // set true if existing sac is in India
							mx_operation_required := false // set true if a change to MX setting is required
							for i := range india_sac_array {
    								if india_sac_array[i] == strconv.Itoa(int(current_terminal_data.SAC)) { 
									// this terminal has mx ops applied
        								mx_operation_applied = true
									break
    								}
							}
							
							// what does the new sac code want ?
							for i := range india_sac_array {
                                                        	if india_sac_array[i] == strconv.Itoa(int(dalData.SAC)) {
									if (orch.debug_enabled) {
										fmt.Printf("mx_operation_required set true for india_sac_array %s \n", india_sac_array[i])
									}
                                                                        // this terminal wants MX filter applied, if is isn't already
                                                                        mx_operation_required = true
									break
                                                                }
							}
							if (mx_operation_applied && mx_operation_required) {
								// no action. Terminal was in India and ist still in India
								if (orch.debug_enabled) {
									fmt.Printf("no ops required was inside, still inside \n")
								}
							} else if (mx_operation_applied && !mx_operation_required) {
								// take action to remove current MX settings for this terminal
								if (orch.debug_enabled) {
									fmt.Printf("SOUTHBOUND Required to remove MX data \n")
								}
								// for each SVN used by this terminal, find the subnets
								var number_of_terminal_svn_subnets int = 0
								subnets_list := make(map[int]string)
                                                        	for _, dal_svn := range dalData.SVNs {
									if (orch.debug_enabled) {
                                                                		fmt.Printf("Known Terminal... DAL Message remove MX filter ops required %d \n", dal_svn.SVN)
									}
									// for each DID entry in the subnets.txt file
                                                                	for  _, terminal_ips := range global_orch.Terminal_IPs_List {
                                                                        	if (terminal_ips.Terminal_did == dalData.DID && terminal_ips.SVN == dal_svn.SVN) {
											// collect all the subnets for this terminal and send as one
											// netconf command
											subnets_list[number_of_terminal_svn_subnets]=terminal_ips.Subnet
											number_of_terminal_svn_subnets++
											if (orch.debug_enabled) {
                                                                                		fmt.Printf("found terminal_ips.Terminal_did %d Subnet %s \n", terminal_ips.Terminal_did, terminal_ips.Subnet)
											}
										}
									}
                                                                        // block or redirect ? check SVN to decide
                                                                        for _, svn_op := range orch.config.SVNlist {
                                                                        	if (svn_op.Name == dal_svn.SVN && svn_op.Operation == "redirect") {
											if (orch.debug_enabled) {
                                                                                                fmt.Printf("dal_svn.SVN %d requires redirect \n",  dal_svn.SVN)
											}
											// syslog
											var syslog_out string = "SAC CONTROLLER licensed SVN " + uint2string(svn_op.Name) + " for Terminal " + uint642string(dalData.DID) + " exiting 12nm of India with SAC code " +  uint2string(dalData.SAC) + "\n"
											fmt.Printf (syslog_out)
											if netconf_operation(orch.config.Devicelist, subnets_list,uint2string(svn_op.Name),false, true) {  // set operation false (delete), licensed true
												if (orch.debug_enabled) {
                                                                                                	fmt.Printf("netconf_operation ok \n")
												}
											}
                                                                                } else if (svn_op.Name == dal_svn.SVN && svn_op.Operation == "block"){
											if (orch.debug_enabled) {
                                                                                                fmt.Printf("dal_svn.SVN %d requires block \n",  dal_svn.SVN)
											}
											// syslog
											var syslog_out string = "SAC CONTROLLER unlicensed SVN " + uint2string(svn_op.Name) + " for Terminal " + uint642string(dalData.DID) + " exiting 12nm of India with SAC code " +  uint2string(dalData.SAC) + "\n"
                                                                                        fmt.Printf(syslog_out)
                                                                                        if netconf_operation(orch.config.Devicelist,subnets_list,uint2string(svn_op.Name),false, false) {// set operation false (delete), licensed false
                                                                                                if (orch.debug_enabled) {
                                                                                                        fmt.Printf("netconf_operation ok \n")
                                                                                                }
                                                                                        }
                                                                                }
                                                                           
                                                                        	
                                                                	}
                                                        	}
							} else if (!mx_operation_applied && !mx_operation_required) {
								// no action
								if (orch.debug_enabled) {
									fmt.Printf("no ops required was outside, still outside \n")
								}
							} else if (!mx_operation_applied && mx_operation_required) {
								// take action to apply current MX settings for this terminal
								if (orch.debug_enabled) {
									fmt.Printf("SOUTHBOUND Required to set MX data \n")
								}
								// for each SVN used by this terminal, find the subnets
								for _, dal_svn := range dalData.SVNs {
									if (orch.debug_enabled) {
                                                                		fmt.Printf("New Terminal... DAL Message  SOUTHBOUND ops required %d \n", dal_svn.SVN)
                                                                		fmt.Printf("global_orch.Terminal_IPs_List len %d \n", len(global_orch.Terminal_IPs_List))
									}
									// for each DID entry in the subnets.txt file
									var number_of_terminal_svn_subnets int = 0
                                                                	subnets_list := make(map[int]string)
                                                                	for  _, terminal_ips := range global_orch.Terminal_IPs_List {
                                                                        	if (terminal_ips.Terminal_did == dalData.DID && terminal_ips.SVN == dal_svn.SVN) {
											subnets_list[number_of_terminal_svn_subnets]=terminal_ips.Subnet
                                                                                        number_of_terminal_svn_subnets++
											if (orch.debug_enabled) {
                                                                                		fmt.Printf("found terminal_ips.Terminal_did %d Subnet %s \n", terminal_ips.Terminal_did, terminal_ips.Subnet)
											}
										}
									}
                                                                        // block or redirect ? check SVN to decide
                                                                        for _, svn_op := range orch.config.SVNlist {
                                                                        	if (svn_op.Name == dal_svn.SVN && svn_op.Operation == "redirect") {
											if (orch.debug_enabled) {
                                                                                        	fmt.Printf("dal_svn.SVN %d requires redirect \n",  dal_svn.SVN)
											}
											// syslog
											var syslog_out string = "SAC CONTROLLER licensed SVN " + uint2string(svn_op.Name) + " for Terminal " + uint642string(dalData.DID) + " entering 12nm of India with SAC code " +  uint2string(dalData.SAC) + "\n"
											fmt.Printf(syslog_out)
                                                                                        if netconf_operation(orch.config.Devicelist, subnets_list,uint2string(svn_op.Name),true, true) { // set operation true licensed true
                                                                                                if (orch.debug_enabled) {
                                                                                                        fmt.Printf("netconf_operation ok \n")
                                                                                                } 
                                                                                        }
                                                                                } else if (svn_op.Name == dal_svn.SVN && svn_op.Operation == "block"){
											if (orch.debug_enabled) {
                                                                                                fmt.Printf("dal_svn.SVN %d requires block \n",  dal_svn.SVN)
											}
											// syslog
											var syslog_out string = "SAC CONTROLLER unlicensed SVN " + uint2string(svn_op.Name) + " for Terminal " + uint642string(dalData.DID) + " entering 12nm of India with SAC code " +  uint2string(dalData.SAC) + "\n"
											fmt.Printf(syslog_out)
                                                                                        if netconf_operation(orch.config.Devicelist, subnets_list, uint2string(svn_op.Name), true, false) {// set operation true, licensed false
                                                                                                if (orch.debug_enabled) {
                                                                                                        fmt.Printf("netconf_operation ok \n")
                                                                                                }
                                                                                        }
                                                                        	}
                                                                	}
                                                        	}
							} 
							// update the table with new dal information
                                                	orch.dALTerminalData[dalData.DID] = dalData
							// backup new data in database
                                                        backupDALData(orch, dalData)
							break
							
						} else {
							// No SAC change. This should never happen because DAL only sends on a sac code change
						  	orch.dALTerminalData[dalData.DID] = dalData
                                               		break
						}
                        		} else {
						if (orch.debug_enabled) {
							fmt.Printf("DAL Message new Terminal %d \n",dalData.DID)
						}
						// ensure the switch is in the right state
						mx_operation_required := false // set true if SAC code is in India
						for i := range india_sac_array {
                                                	if india_sac_array[i] ==  strconv.Itoa(int(dalData.SAC)) { 
                                                        	// this terminal is operating in India SAC
                                                                mx_operation_required = true
                                                                break
                                                        }
                                                }
						if (mx_operation_required) {
							// action 1
							if (orch.debug_enabled) {
								fmt.Printf("New Terminal... DAL Message  SOUTHBOUND ops required dalData.SVNs %d \n", len (dalData.SVNs))
							}
							// For each SVN used by this terminal
							var number_of_terminal_svn_subnets int = 0
                                                        subnets_list := make(map[int]string)
							for _, dal_svn := range dalData.SVNs {
								if (orch.debug_enabled) {
									fmt.Printf("New Terminal... DAL Message  SOUTHBOUND ops required %d \n", dal_svn.SVN)
									fmt.Printf("global_orch.Terminal_IPs_List len %d \n", len(global_orch.Terminal_IPs_List))
								}
								// Lookup subnets by DID and SVN
								for  _, terminal_ips := range global_orch.Terminal_IPs_List {
									if (terminal_ips.Terminal_did == dalData.DID && terminal_ips.SVN == dal_svn.SVN) {
										subnets_list[number_of_terminal_svn_subnets]=terminal_ips.Subnet
                                                                                number_of_terminal_svn_subnets++
										if (orch.debug_enabled) {
											fmt.Printf("found terminal_ips.Terminal_did %d Subnet %s \n", terminal_ips.Terminal_did, terminal_ips.Subnet)
										}
									}
								}
								// block or redirect ? check SVN to decide
								for _, svn_op := range orch.config.SVNlist {
									if (svn_op.Name == dal_svn.SVN && svn_op.Operation == "redirect") {
										if (orch.debug_enabled) {
											fmt.Printf("dal_svn.SVN %d requires redirect \n",  dal_svn.SVN)
										}
										// syslog
										var syslog_out string = "SAC CONTROLLER licensed SVN " + uint2string(svn_op.Name) + " for Terminal " + uint642string(dalData.DID) + " currently 12nm of India with SAC code " +  uint2string(dalData.SAC) + "\n"
										fmt.Printf(syslog_out)
                        							if netconf_operation(orch.config.Devicelist,subnets_list, uint2string(svn_op.Name),true, true) { // set operation true, license is true
                                                                                         if (orch.debug_enabled) {
                                                                                         	fmt.Printf("netconf_operation ok \n")
                                                                                         }
 
                                                                                }
									} else if (svn_op.Name == dal_svn.SVN && svn_op.Operation == "block"){
										if (orch.debug_enabled) {
											fmt.Printf("dal_svn.SVN %d requires block \n",  dal_svn.SVN)
										}
										// syslog 
										var syslog_out string = "SAC CONTROLLER unlicensed SVN " + uint2string(svn_op.Name) + " for Terminal " + uint642string(dalData.DID) + " currently 12nm of India with SAC code " +  uint2string(dalData.SAC) + "\n"
										fmt.Printf(syslog_out)
                                                                                if netconf_operation(orch.config.Devicelist,subnets_list,uint2string(svn_op.Name),true, false) { // set operation true, license is false
                                                                                         if (orch.debug_enabled) {
                                                                                                fmt.Printf("netconf_operation ok \n")
                                                                                         }

                                                                                }
									}
								}
							}			
						} else {
							if (orch.debug_enabled) {
								fmt.Printf("DAL Message SAC code outside India - no operation required \n")
							}
						}
						orch.dALTerminalData[dalData.DID] = dalData
						backupDALData(orch, dalData)
						break
					}
					
				}
		}
	}
}

func CreateStackTrace (orch *Orchestrator) {

	defer func() {
		debug.PrintStack()
	} ()
}

func main() {
    var orch Orchestrator
    orch.Configure()
    restoreDALData(&orch)
    // const time for influx db writes
    date := "2000-12-31"
    const_time, _ = time.Parse(layoutISO, date)
    go startAutomation(&orch)
    LoadSubnets(&orch)
    flag.Parse()
    startListeningForHTTP(&orch)
    fmt.Printf("main program exit  \n")
    os.Exit(mainReturnWithCode())
}
