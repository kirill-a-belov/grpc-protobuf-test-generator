package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"text/template"
	"time"
)

type Config struct {
	Name    string
	NameF   string
	MethodF string
	Address string
	Request string
	File    string
	Random  string
	Path    string
	Save    bool
}

var config Config

const TEMPLATE = `package main

import (
	"{{.Path}}"
	"google.golang.org/grpc"
	"fmt"
	"encoding/json"
	"context"
)

var client {{.Name}}Rpc.{{.NameF}}ApiClient

func main() {
	var opts []grpc.DialOption
	var err error

	opts = append(opts, grpc.WithInsecure())
	conn, err := grpc.Dial("{{.Address}}", opts...)
	if err != nil {
		fmt.Println("fail to dial:  ", err)
	}
	defer conn.Close()
	client = {{.Name}}Rpc.New{{.NameF}}ApiClient(conn)


	//************************** All tests are here *******************************************************

	//Test template

	fmt.Printf("\n\n************************** Test for \"{{.MethodF}}\" has started **************************\n\n")
	request := &{{.Name}}Rpc.{{.Request}}

	jsnb, err := json.MarshalIndent(request, "", "\t")
	if err != nil {
		panic(err)
	}
	fmt.Println("Request is:")
	fmt.Println(string(jsnb))


	fmt.Printf("\n\nResponse is:\n")
	response, err := client.{{.MethodF}}(context.Background(), request)
	if response != nil {
		jsnb, err := json.MarshalIndent(response, "", "\t")
		if err != nil {
			panic(err)
		}
		fmt.Println(string(jsnb))
	}
	if err != nil {
    	fmt.Println("Error:  ", err)
    	fmt.Println("Test NOT PASSED!")
    } else {
        fmt.Println("Test IS PASSED!")
    }


	fmt.Printf("************************** Test for \"{{.MethodF}}\" has finished **************************\n\n")

}`

func init() {
	flag.StringVar(&config.Address, "address", "", "server address (eg. 127.0.0.1:7015)")
	flag.StringVar(&config.File, "file", "", "name of proto file (eg. server-grpc.proto)")
	flag.StringVar(&config.MethodF, "method", "", "method for test (eg. DoSomeWork)")
	flag.BoolVar(&config.Save, "save", false, "optional flag if you want to save temp dir with tests")

	flag.Parse()

	config.Name = config.File
	if strings.HasSuffix(config.Name, ".proto") {
		config.Name = strings.TrimSuffix(config.Name, ".proto")
	}
	if strings.HasSuffix(config.Name, "-grpc") {
		config.Name = strings.TrimSuffix(config.Name, "-grpc")
	}

	config.NameF = strings.ToUpper(string(config.Name[0])) + config.Name[1:]
	config.MethodF = strings.ToUpper(string(config.MethodF[0])) + config.MethodF[1:]
	config.Random = randomString() + "_test"
	//Getting path
	config.Path, _ = os.Getwd()
	goSrcPath := os.Getenv("GOPATH") + "/src/"

	config.Path = strings.TrimPrefix(config.Path, goSrcPath)

	config.Path = config.Path + "/" + config.Random + "/" + config.Name + "Rpc"

	if config.Address == "" || config.Name == "" || config.MethodF == "" {
		fmt.Println("problem with params: ", config.Address, config.Name, config.MethodF)
	}
}

func main() {
	//First - compile proto file
	cmd := exec.Command("mkdir", config.Random)
	err := cmd.Run()
	if err != nil {
		fmt.Println("error while creating directory")
		panic(err)
	}
	cmd = exec.Command("mkdir", config.Random+"/"+config.Name+"Rpc")
	err = cmd.Run()
	if err != nil {
		fmt.Println("error while creating directory")
		panic(err)
	}
	cmd = exec.Command("protoc", config.File, "--go_out=plugins=grpc:./"+config.Random+"/"+config.Name+"Rpc")
	err = cmd.Run()
	if err != nil {
		fmt.Println("error while proto compilating")
		panic(err)
	}
	//Find proper request for method
		//Read proto file to string
	buf := bytes.NewBuffer(nil)

	f, err := os.Open(config.File)
	if err != nil {
		fmt.Println("error while proto file reading")
		panic(err)
	}
	io.Copy(buf, f)
	f.Close()

	protoText := string(buf.Bytes())

		//Getting request name for method
	var re = regexp.MustCompile("rpc *" + config.MethodF + " *(.*) *returns")
	reqName := re.FindString(protoText)
	reqName = strings.Replace(reqName, " ", "", -1)
	reqName = strings.TrimPrefix(reqName, "rpc"+config.MethodF+"(")
	reqName = strings.TrimSuffix(reqName, ")returns")

		//Getting request message
	var re2 = regexp.MustCompile("message *" + reqName + " {([^}]*)")
	var re3 = regexp.MustCompile(` *=.*`)
	reqContain := re2.FindString(protoText)

	reqContains := strings.Split(reqContain, "\n")
	reqContains = reqContains[1:]

	fmt.Println("We need some data for request, please input:")
	config.Request = config.Request + reqName + "{"
	for _, rc := range reqContains {
		if len(rc) < 10 {
			continue
		}
		rc = re3.ReplaceAllLiteralString(rc, "")
		fmt.Println(rc + ":")
		input := ""
		_, err := fmt.Scanln(&input)
		if err != nil {
			fmt.Println("error while scaning varible", rc)
			panic(err)
		}
		if rc[4] == 's' {
			config.Request = config.Request + "\"" + input + "\", "
		} else {
			config.Request = config.Request + input + ", "
		}
	}
	config.Request = config.Request[:len(config.Request)-2] + "}"

	//Fill template by varibles
	tmpl, err := template.New("test").Parse(TEMPLATE)
	if err != nil {
		fmt.Println("error while parsing template")
		panic(err)
	}

	buf = bytes.NewBufferString("")
	err = tmpl.Execute(buf, config)
	if err != nil {
		fmt.Println("error while template filling")
		panic(err)
	}

	//Saving to go file
	outFile, err := os.Create("./" + config.Random + "/test.go")
	if err != nil {
		fmt.Println("error while creating output file")
		panic(err)
	}

	outFile.Write(buf.Bytes())

	//Runing this new file

	cmd = exec.Command("go", "run", "./"+config.Random+"/test.go")
	cmd.Stdout = os.Stdout
	err = cmd.Run()
	if err != nil {
		fmt.Println("error while running test.go")
		panic(err)
	}

	//If we have empty save flag - remove directory

	if !config.Save {
		err := os.RemoveAll(config.Random)
		if err != nil {
			fmt.Println("error while rcleaning temp dir")
			panic(err)
		}
	}

}

func randomString() string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, 10)
	for i := range result {
		result[i] = chars[r.Intn(len(chars))]
	}
	return string(result)
}
