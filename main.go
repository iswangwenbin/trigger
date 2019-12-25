package main

import (
	"fmt"
	"github.com/micro/go-micro/config"
	cryptoSSH "golang.org/x/crypto/ssh"
	"gopkg.in/go-playground/webhooks.v5/gitlab"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	gitHTTP "gopkg.in/src-d/go-git.v4/plumbing/transport/http"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

var hook *gitlab.Webhook

func init() {
	loadConfig("")
	hook, _ = gitlab.New()
}

func main() {
	http.HandleFunc("/api", trigger)
	port := config.Get("ServiceSettings", "ListenAddress").String(":9000")
	http.ListenAndServe(port, nil)
}

func trigger(w http.ResponseWriter, r *http.Request) {
	payload, err := hook.Parse(r, gitlab.PushEvents)
	if err != nil {
		if err == gitlab.ErrEventNotFound {
			log.Fatal(err)
		}
	}
	switch payload.(type) {
	case gitlab.PushEventPayload:
		push := payload.(gitlab.PushEventPayload)
		fmt.Printf("======== PushEvent ========\n")
		fmt.Printf("Name:%+v\n", push.Repository.Name)
		fmt.Printf("URL:%+v\n", push.Repository.URL)
		path := config.Get("Repositories", push.Repository.Name).String("")
		pullCode(path)
	default:
		fmt.Printf("%+v", payload)
	}
}

func pullCode(path string) {
	if path == "" {
		log.Print("path is not exists")
		return
	}
	r, err := git.PlainOpen(path)
	if err != nil {
		log.Print(err)
		return
	}
	w, err := r.Worktree()
	if err != nil {
		log.Print(err)
		return
	}
	err = w.Pull(&git.PullOptions{
		RemoteName: "origin",
		Auth: &gitHTTP.BasicAuth{
			Username: config.Get("Git", "Username").String(""),
			Password: config.Get("Git", "Password").String(""),
		},
		Progress: os.Stdout,
	})
	if err != nil {
		log.Print(err)
		return
	}
	ref, err := r.Head()
	if err != nil {
		log.Print(err)
		return
	}
	commit, err := r.CommitObject(ref.Hash())
	if err != nil {
		log.Print(err)
		return
	}
	fmt.Println(commit)
}

func getSSHKeyAuth(privateSshKeyFile string) transport.AuthMethod {
	var auth transport.AuthMethod
	sshKey, _ := ioutil.ReadFile(privateSshKeyFile)
	signer, _ := cryptoSSH.ParsePrivateKey([]byte(sshKey))
	auth = &ssh.PublicKeys{User: "git", Signer: signer}
	return auth
}

func loadConfig(path string) {
	if path == "" {
		path = "config.json"
	}
	err := config.LoadFile(path)
	if err != nil {
		panic(err)
	}
}

