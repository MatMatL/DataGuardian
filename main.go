package main

//export PATH=$PATH:/usr/local/go/bin

import (
	"bufio"
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

const port = ":8080"

const USER = "mathieu"
const DESTINATION = "~/backup/"

var index = template.Must(template.ParseFiles("index.html"))
var menu = template.Must(template.ParseFiles("menu.html"))
var automatisation = template.Must(template.ParseFiles("automatisation.html"))

func main() {
	http.HandleFunc("/", Index)
	http.HandleFunc("/menu", Menu)
	http.HandleFunc("/automatisation", Automatisation)

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	fmt.Println("Server started on port : ", port)

	err := exec.Command("xdg-open", "http://localhost:8080").Start()
	if err != nil {
		fmt.Println("erreur lors de l'ouverture web : ", err)
	}
	http.ListenAndServe(port, nil)
}

type toSend struct {
	Path    string
	IP      string
	Action  string
	Message string
}

var ToSend = toSend{}

func Index(w http.ResponseWriter, r *http.Request) {
	if ToSend.IP != "" && ToSend.Path != "" {
		http.Redirect(w, r, "/menu", http.StatusSeeOther)
		return
	}

	if r.Method == "POST" {
		r.ParseForm()

		ToSend.IP = r.FormValue("Ip")
		ToSend.Path = r.FormValue("Path")
		fmt.Println(ToSend.Path[len(ToSend.Path)-1])

		fmt.Println(ToSend.IP, ToSend.Path)

		if ToSend.IP != "" && ToSend.Path != "" {
			http.Redirect(w, r, "/menu", http.StatusSeeOther)
			return
		}
	}
	index.ExecuteTemplate(w, "index.html", ToSend)
}

func Menu(w http.ResponseWriter, r *http.Request) {
	if ToSend.IP == "" || ToSend.Path == "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if r.Method == "POST" {
		r.ParseForm()

		NewIP := r.FormValue("IP")
		if NewIP != "" {
			ToSend.IP = NewIP
		}

		NewPath := r.FormValue("Path")
		if NewPath != "" {
			ToSend.Path = NewPath
		}

		action := r.FormValue("action")

		switch action {
		case "upload":
			Confirm(w, r, "upload")
			return
		case "restore":
			Confirm(w, r, "restore")
			return
		}
	}

	menu.ExecuteTemplate(w, "menu.html", ToSend)
}

func Confirm(w http.ResponseWriter, r *http.Request, action string) {
	serverAdresse := USER + "@" + ToSend.IP + ":" + DESTINATION
	var commande *exec.Cmd
	fmt.Println(commande)
	if ToSend.Path[len(ToSend.Path)-1] == 47 {
		ToSend.Path = ToSend.Path[:len(ToSend.Path)-1]
	}
	if action == "upload" {
		commande = exec.Command("rsync", "-avz", "--delete", "-e", "ssh -i ~/SiInfra/key.txt", ToSend.Path, serverAdresse)
	} else if action == "restore" {
		ServerAdressePlus := serverAdresse + GetLastPath()
		commande = exec.Command("rsync", "-avz", "--delete", "-e", "ssh -i ~/SiInfra/key.txt", ServerAdressePlus, ToSend.Path)
	}

	fmt.Println("Command executed : ", commande)

	output, err := commande.Output()
	if err != nil {
		fmt.Println(err)

		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode := exitError.ExitCode()
			if exitCode == 23 {
				ToSend.Message = "Dossier introuvable, veuillez verifier le chemin d'accès"
			} else {
				ToSend.Message = "Adresse Ip du serveur distant incorrecte"
			}
		}
		menu.ExecuteTemplate(w, "menu.html", ToSend)
		return
	}

	fmt.Println(string(output))

	if action == "upload" {
		ToSend.Message = "La sauvegarde s'est réalisée avec succès"
	} else if action == "restore" {
		ToSend.Message = "La restauration s'est réalisée avec succès"
	}

	menu.ExecuteTemplate(w, "menu.html", ToSend)
}

func GetLastPath() string {
	PathPieces := strings.Split(ToSend.Path, "/")
	return PathPieces[len(PathPieces)-1] + "/"
}

func Automatisation(w http.ResponseWriter, r *http.Request) {
	if ToSend.IP == "" || ToSend.Path == "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	if r.Method == "POST" {
		r.ParseForm()

		CrontabCommand := r.FormValue("Crontab")

		RefleshShFile()

		NewCrontab(CrontabCommand + " ~/SiInfra/backup.sh > ~/backup.log 2>&1")
	}

	automatisation.ExecuteTemplate(w, "automatisation.html", ToSend)
}

func NewCrontab(cronJob string) error {
	fmt.Println("New crontab input :" + cronJob)
	cmd := exec.Command("crontab", "-")
	cmd.Stdin = bytes.NewBufferString(cronJob + "\n")
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("erreur lors de la mise à jour de la crontab : %v", err)
	}

	return nil
}

func RefleshShFile() {
	script, err := readScript("./backup.sh")
	if err != nil {
		fmt.Println(err)
		return
	}

	script = replaceVariable(script, "SOURCE", ToSend.Path)
	script = replaceVariable(script, "HOST", ToSend.IP)

	err = writeScript("./backup.sh", script)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("Le fichier de script a été mis à jour avec succès.")

	fmt.Println("Le fichier de script a été mis à jour avec succès.")
}

func replaceVariable(script, variable, newValue string) string {
	re := regexp.MustCompile(fmt.Sprintf(`(?m)^%s="[^"]*"`, variable))
	return re.ReplaceAllString(script, fmt.Sprintf(`%s="%s"`, variable, newValue))
}

func readScript(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("erreur lors de l'ouverture du fichier: %v", err)
	}
	defer file.Close()

	var script string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		script += scanner.Text() + "\n"
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("erreur lors de la lecture du fichier: %v", err)
	}
	return script, nil
}

func writeScript(filePath, script string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("erreur lors de la création du fichier: %v", err)
	}
	defer file.Close()

	_, err = file.WriteString(script)
	if err != nil {
		return fmt.Errorf("erreur lors de l'écriture dans le fichier: %v", err)
	}
	return nil
}
