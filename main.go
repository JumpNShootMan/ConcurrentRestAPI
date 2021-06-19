package main

//Ejecución del RestAPI
//go mod init github.com/Username/NombreCarpeta
//go mod tidy
//go build github.com/Username/NombreCarpeta
//./NombreCarpeta
//puerto es 8080
//el front corre desde svelte en el puerto 5000
//npx degit sveltejs/template my-svelte-project para generar el front svelte
//no olvidar npm install
//npm run dev para correr front
//estilos de front investigados: https://bootswatch.com/cyborg/

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

type barrier struct { //posible forma de aplicar concurrencia
	count  int
	n_iter int
	mu     sync.Mutex
	signal chan int
	wait   chan int
}

//JSON_Input es el punto a recibir
type JSON_Input struct {
	X float64 `json:"x"` // posicion x en knn
	Y float64 `json:"y"` // posicion y en knn
	K []byte  `json:"k"` // num vecinos en knn
}

//salida JSON
type JSON_Output struct {
	Data    []Data     `json:"data"`
	Caminos [][]Labels `json:"caminos"`
	Clases  []string   `json:"clases"`
}

var retorno JSON_Output

//Puntos x, y, clase para el plano cartesiano en el que trabaja KNN
type Punto struct {
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Clase string  `json:"clase"` //clase
}

type Labels struct {
	Nombre string `json:"nombre"`
	Conteo int    `json:"conteo"`
}

//impresión de los resultados
func (p Punto) String() string {
	return fmt.Sprintf("X = %f, Y = %f es: Fabricante = %s\n", p.X, p.Y, p.Clase)
}

type Data struct {
	Punto        Punto   `json:"punto"`        // punto de coordenadas y clase para knn
	Distancia    float64 `json:"distancia"`    // distancia euclideana del punto
	Departamento string  `json:"departamento"` // departamento del sujeto
	Distrito     string  `json:"distrito"`     // distrito del sujeto
	Grupo_Riesgo string  `json:"grupo_riesgo"` // grupo de riesgo descripción
}

func (d Data) String() string {
	return fmt.Sprintf(
		"X = %f Y = %f, Distancia Euclideana = %f Clase = %s, Departamento = %s, Distrito = %s, Grupo_Riesgo= %s\n",
		d.Punto.X, d.Punto.Y, d.Distancia, d.Punto.Clase, d.Departamento, d.Distrito, d.Grupo_Riesgo,
	)
}

//Se requiere procesar facilmente información para obtener la longitud, para intercambiar rápidamente datos y para facilitar condicionales
type Block []Data

//Creación rápida de funciones
func (b Block) Len() int           { return len(b) }
func (b Block) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b Block) Less(i, j int) bool { return b[i].Distancia < b[j].Distancia }

//Función para distancia euclideana
func DEuclidiana(A Punto, X Punto) (distancia float64, err error) {
	distancia = math.Sqrt(math.Pow((X.X-A.X), 2) + math.Pow((X.Y-A.Y), 2))
	if distancia < 0 {
		return 0, fmt.Errorf("distancia euclideana negativa, datos inválidos")
	}

	return distancia, nil
}

func readCSVFromUrl(url string) ([][]string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	reader := csv.NewReader(resp.Body)
	data, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	return data, nil
}

func LoadData(csvPath string) (data []Data, err error) {
	fd, err := os.Open(csvPath)
	if err != nil {
		return nil, err
	}
	reader := csv.NewReader(fd)
	datoscsv, _ := reader.ReadAll()
	fmt.Println("Carga de datos\n")
	filas := len(datoscsv)
	columnas := len(datoscsv[0])

	for i := 0; i < filas; i++ {
		for j := 0; j < columnas; j++ {
			fmt.Printf("%s\t  ", datoscsv[i][j])
		}
		if i == 0 {
			fmt.Println()
		}
		fmt.Println()
	}
	fmt.Println()
	var valor float64
	data = make([]Data, filas-1)
	for i := 0; i < filas-1; i++ {

		//archivos flotantes requieren una conversión para pasar como tal
		valor, err = strconv.ParseFloat(datoscsv[i+1][0], 64)
		if err != nil {
			return nil, fmt.Errorf("error en parse para X con valor: %v", err)
		}
		data[i].Punto.X = valor //float

		valor, err = strconv.ParseFloat(datoscsv[i+1][1], 64)
		if err != nil {
			return nil, fmt.Errorf("error en parse para Y con valor: %v", err)
		}
		data[i].Punto.Y = valor //float

		data[i].Punto.Clase = datoscsv[i+1][2] //string

		data[i].Departamento = datoscsv[i+1][3] //string

		data[i].Distrito = datoscsv[i+1][4] //string

		//agregar por cantidad de columnas
		data[i].Grupo_Riesgo = datoscsv[i+1][5] //string

	}
	return data, nil
}

// func ValidError(err error) {
// 	if err != nil {
// 		fmt.Printf("[!] %s\n", err.Error())
// 		os.Exit(1)
// 	}
// }

func Knn(data []Data, k byte, X *Punto) (err error) {
	n := len(data)
	// calcular distancias
	for i := 0; i < n; i++ {
		if data[i].Distancia, err = DEuclidiana(data[i].Punto, *X); err != nil {
			return err
		}
	}

	var blk Block = data
	// ordenar ascendiendo
	sort.Sort(blk)
	var save []Labels
	// pass
	if int(k) > n {
		return nil
	}
	for i := byte(0); i < k; i++ {
		save = IncrementoLabels(data[i].Punto.Clase, save)
	}

	fmt.Printf("[*] Using k as %d\n", k)
	fmt.Println()
	fmt.Printf("[*] %+v\n", save)
	fmt.Println()

	retorno.Caminos = append(retorno.Caminos, save)

	max := 0
	var maxLabel string
	m := len(save)
	for i := 0; i < m; i++ {
		if max < save[i].Conteo {
			max = save[i].Conteo
			maxLabel = save[i].Nombre
		}
	}

	X.Clase = maxLabel
	retorno.Clases = append(retorno.Clases, maxLabel)
	return nil
}

func IncrementoLabels(label string, labels []Labels) []Labels {
	if labels == nil {
		labels = append(labels, Labels{
			Nombre: label,
			Conteo: 1,
		})
		return labels
	}

	conteo := len(labels)
	for i := 0; i < conteo; i++ {
		if strings.Compare(labels[i].Nombre, label) == 0 {
			labels[i].Conteo++
			return labels
		}
	}

	return append(labels, Labels{
		Nombre: label,
		Conteo: 1,
	})
}

func API_KNN(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Counter-Type", "application/json")

	data, _ := LoadData("vacunas_covid.csv")
	// ValidError(err)

	// Queremos leer desde un JSON
	var json_input JSON_Input
	_ = json.NewDecoder(r.Body).Decode(&json_input)
	var X Punto
	X.X = json_input.X
	X.Y = json_input.Y
	var k = json_input.K

	n := len(k)
	for i := 0; i < n; i++ {
		_ = Knn(data, k[i], &X)
		if i == 0 {
			fmt.Println(data)
			retorno.Data = data
		}
		// ValidError(err)
		fmt.Printf("La clase dominante para la proximidad de los siguientes datos: ")
		fmt.Println(X)
	}

	json.NewEncoder(w).Encode(retorno)
	var aux JSON_Output
	retorno = aux
}

func main() {
	//Creación de router
	r := mux.NewRouter()
	var wg sync.WaitGroup
	wg.Add(2)
	//CORS para poder trabajar con interfaz web
	//Enlace para API
	go func() {
		r.HandleFunc("/api/knn", API_KNN).Methods("POST")
		wg.Done()
	}()

	go func() {

		log.Fatal(
			http.ListenAndServe(
				":8080",
				handlers.CORS(
					handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Authorization"}),
					handlers.AllowedMethods([]string{"GET", "POST", "PUT", "HEAD", "OPTIONS"}),
					handlers.AllowedOrigins([]string{"*"}))(r)))
		wg.Done()
	}()

	wg.Wait()
}
