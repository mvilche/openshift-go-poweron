package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/mgo.v2/bson"
)

/// define modelo en mongo
type Microservicios struct {
	Microservicio         string `bson:"Microservicio"`
	Replicas              string `bson:"Replicas"`
	Porcentaje            int64  `bson:"Porcentaje"`
	ReplicasConPorcentaje int64  `bson:"ReplicasConPorcentaje"`
	Date                  string `bson:"Date"`
	Hora                  string `bson:"Hora"`
	Usuario               string `bson:"Usuario"`
	Namespace             string `bson:"Namespace"`
}

type MicroserviciosScale struct {
	Microservicio string `bson:"Microservicio"`
	Replicas      string `bson:"Replicas"`
	Date          string `bson:"Date"`
	Hora          string `bson:"Hora"`
	Usuario       string `bson:"Usuario"`
	Namespace     string `bson:"Namespace"`
}

/// define estructura en datasource
type Config struct {
	Dbpassword         string
	DbDatabase         string
	DbUser             string
	DbPort             int64
	DbHost             string
	OpenshiftHost      string
	OpenshiftUser      string
	OpenshiftPassword  string
	OpenshiftNamespace string
	DbCollection       string
	DbCollectionInsert string
}

/// lee datasource

func empty(s string) bool {
	return len(strings.TrimSpace(s)) == 0
}

func ReadConfig() Config {

	if _, err := os.Stat("libs"); os.IsNotExist(err) {
		log.Fatal("ERROR NO SE ENCONTRO CARPETA libs")
		os.Exit(1)
	}

	if _, err := os.Stat("libs/oc"); os.IsNotExist(err) {
		log.Fatal("ERROR NO SE ENCONTRO BINARIO oc EN CARPETA libs \n DESCARGUE https://github.com/openshift/origin/releases/download/v3.11.0/openshift-origin-client-tools-v3.11.0-0cbc58b-linux-64bit.tar.gz")
		os.Exit(1)
	}

	if _, err := os.Stat("config"); os.IsNotExist(err) {
		log.Fatal("ERROR NO SE ENCONTRO CARPETA config")
		os.Exit(1)
	}

	var configfile = "config/config.conf"
	_, err := os.Stat(configfile)
	if err != nil {
		log.Fatal("ERROR NO SE ENCONTRO DATASOURCE: ", configfile)
		os.Exit(1)
	}

	var config Config
	if _, err := toml.DecodeFile(configfile, &config); err != nil {
		log.Fatal("ERROR AL COMPROBAR EL ARCHIVO - CONTIENE ERRORES", err)
	}

	if empty(config.DbDatabase) {
		log.Fatal("ERROR dbdatabase vacio en config.conf")
		os.Exit(1)
	}

	if empty(config.DbHost) {
		log.Fatal("ERROR dbhost vacio en config.conf")
		os.Exit(1)
	}

	s := strconv.FormatInt(int64(config.DbPort), 10)

	if empty(s) {
		log.Fatal("ERROR dbport vacio en config.conf")
		os.Exit(1)
	}

	if empty(config.OpenshiftHost) {
		log.Fatal("ERROR openshifthost vacio en config.conf")
		os.Exit(1)

	}

	if empty(config.OpenshiftUser) {
		log.Fatal("ERROR openshiftuser vacio en config.conf")
		os.Exit(1)

	}

	if empty(config.OpenshiftPassword) {
		log.Fatal("ERROR openshiftpassword vacio en config.conf")
		os.Exit(1)

	}

	if empty(config.OpenshiftNamespace) {
		log.Fatal("ERROR openshiftnamespace vacio en config.conf")
		os.Exit(1)

	}

	return config
}

//// comprueba mongo
func Mongoconnect() {

	host := ReadConfig().DbHost
	port := ReadConfig().DbPort
	clientOpts := options.Client().ApplyURI(fmt.Sprintf("mongodb://%s:%d", host, port))
	client, err := mongo.Connect(context.TODO(), clientOpts)
	if err != nil {
		fmt.Println("ERROR CONECTANDO A MONGO: ")
		log.Fatal(err)
		os.Exit(1)
	}

	// Check the connections
	err = client.Ping(context.TODO(), nil)
	if err != nil {
		fmt.Println("ERROR CONECTANDO A MONGO: ")
		log.Fatal(err)
		os.Exit(1)
	}

	err = client.Disconnect(context.TODO())

	if err != nil {
		log.Fatal(err)
	}

}

func OpenshiftGetReplicas() {
	/*
		namespace := ReadConfig().OpenshiftNamespace
		usuario := ReadConfig().OpenshiftUser
	*/
	host := ReadConfig().DbHost
	database := ReadConfig().DbDatabase
	port := ReadConfig().DbPort
	DbCollectionInsert := ReadConfig().DbCollectionInsert
	DbCollection := ReadConfig().DbCollection
	Mongoconnect()
	clientOpts := options.Client().ApplyURI(fmt.Sprintf("mongodb://%s:%d", host, port))
	client, err := mongo.Connect(context.TODO(), clientOpts)
	if err != nil {
		fmt.Println("ERROR CONECTANDO A MONGO: ")
		log.Fatal(err)
		os.Exit(1)
	}

	/////////////
	now := NowDate()
	fmt.Println("OBTENIENDO DATOS ALMACENADOS EN BASE\n")
	fmt.Println("BUSCANDO MICROSERVICIOS CON FECHA: ", now, "\n")

	collection := client.Database(database).Collection(DbCollection)
	ctx := context.Background()
	microResult := []Microservicios{}
	n := Microservicios{}

	//cursor, err := collection.Find(ctx, bson.M{})
	cursor, err := collection.Find(ctx, bson.M{"Date": now})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Iterate through the returned cursor.
	for cursor.Next(ctx) {
		cursor.Decode(&n)
		microResult = append(microResult, n)
	}

	if len(microResult) <= 0 {

		fmt.Printf("NO SE ENCONTRARON DATOS EN MONGO")
		os.Exit(1)
	}

	for _, el := range microResult {

		// obtengo fecha de modificacion
		/*	date, err := time.Parse("2006-01-02 15:04:05", el.Date)
			if err != nil {
				fmt.Println(err)
				return
			}
			format := date.Format("2006-01-02")*/
		fmt.Printf("*********************\n")
		fmt.Println("Microservicio: ", el.Microservicio, "\nReplicas actuales: ", el.ReplicasConPorcentaje, "\nReplicas deseadas: ", el.Replicas, "\n")
		fmt.Println("AUMENTANDO CANTIDAD DE REPLICAS DE ", el.Microservicio, " a ", el.Replicas)

		micros := MicroserviciosScale{
			Microservicio: el.Microservicio,
			Replicas:      el.Replicas,
			Date:          NowDate(),
			Hora:          NowHora(),
			Usuario:       el.Usuario,
			Namespace:     el.Namespace,
		}

		cmd := exec.Command("bash", "-c", "libs/oc scale dc/"+el.Microservicio+" --replicas="+el.Replicas+" -n "+el.Namespace)
		cmd.Wait()
		out, err := cmd.CombinedOutput()
		if err != nil {
			out2 := string([]byte(out))
			fmt.Println(out2)
			log.Fatalf("ERROR AL REALIZAR SCALE EN OPENSHIFT %s\n", err)
			fmt.Println("********************************************")
			os.Exit(1)
		}
		salida := string([]byte(out))
		fmt.Println("REALIZANDO SCALE...\n", salida)

		collection2 := client.Database(database).Collection(DbCollectionInsert)
		resultado, err := collection2.InsertOne(context.TODO(), micros)
		if err != nil {
			log.Fatal(err)
		}
		resultado = nil

		fmt.Println("AUMENTO DE REPLICAS CORRECTO", resultado)

	}

}

func NowDate() string {

	current := time.Now()
	format := current.Format("2006-01-02")
	return format
}

func NowHora() string {

	current := time.Now()
	format := current.Format("15:04:05")
	return format
}

// openshift

func CheckOpenshiftNamespace() {

	namespace := ReadConfig().OpenshiftNamespace

	cmd := exec.Command("bash", "-c", "libs/oc get project "+namespace)
	cmd.Wait()
	out, err := cmd.CombinedOutput()
	if err != nil {
		out2 := string([]byte(out))
		fmt.Println(out2)
		log.Fatalf("FALLO AL OBTENER EL NAMESPACE EN OPENSHIFT %s\n", err)
		fmt.Println("********************************************")
		os.Exit(1)
	}

	fmt.Println("NAMESPACE VALIDO EN OPENSHIFT: " + namespace + "\n")
	out3 := string([]byte(out))
	fmt.Println(out3)
	fmt.Println("********************************************")

}

func CheckOpenshiftLogin() {

	user := ReadConfig().OpenshiftUser
	password := ReadConfig().OpenshiftPassword
	host := ReadConfig().OpenshiftHost

	cmd := exec.Command("bash", "-c", "libs/oc login "+host+" --username="+user+" --password="+password+" --insecure-skip-tls-verify=false")
	cmd.Wait()
	_, err := cmd.Output()
	if err != nil {
		fmt.Println("********************************************")
		log.Fatalf("FALLO CONEXION CON OPENSHIFT %s\n", err)
		fmt.Println("********************************************")
		os.Exit(1)
	}

	cmd2 := exec.Command("bash", "-c", "libs/oc whoami -t")
	cmd.Wait()
	out, err := cmd2.Output()
	if err != nil {
		fmt.Println("********************************************")
		log.Fatalf("FALLO CONEXION CON OPENSHIFT %s\n", err)
		fmt.Println("********************************************")
		os.Exit(1)
	}
	auth := string([]byte(out))
	fmt.Println("AUTENTICADO CORRECTAMENTE EN OPENSHIFT TOKEN GENERADO: ", auth)
	fmt.Println("********************************************")
}
