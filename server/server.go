/*
=====================================================================================================
  - Server.go : Deverá consumir a API contendo o câmbio de Dólar e Real no endereço:
  - https://economia.awesomeapi.com.br/json/last/USD-BRL e em seguida deverá retornar no
  - formato JSON o resultado para o cliente.
    -
  - Usando o package "context", deverá registrar no banco de dados SQLite cada cotação
  - recebida, sendo que o timeout máximo para chamar a API de cotação do dólar deverá ser
  - de 200ms e o timeout máximo para conseguir persistir os dados no banco deverá ser de
  - 10ms.
    -
  - O endpoint necessário gerado pelo server.go para este desafio será: /cotacao e a porta
  - a ser utilizada pelo servidor HTTP será a 8080.

=====================================================================================================
*/
package main

import (
	"context"
	"database/sql" //Pacote do driver DB do mysql
	"encoding/json"

	//"fmt"
	"io"
	"log"
	"net/http"

	//"os"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3" //Pacote para usar o SQLite
)

// Estrutura que receberá o parse do JSON
/*type USDBRL struct {
	Dados Data
}
type Data struct {
	Code        string `json:"code"`
	Codein      string `json:"codein"`
	Name        string `json:"name"`
	High        string `json:"high"`
	Low         string `json:"low"`
	VarBid      string `json:"varBid"`
	PctChange   string `json:"pctChange"`
	Bid         string `json:"bid"`
	Ask         string `json:"ask"`
	Timestamp   string `json:"timestamp"`
	Create_date string `json:"create_date"`
}*/

// Retorna o JSON com a cotação do dolar para o client
type Cotacao struct {
	Dolar string `json:"dolar"`
	//Valor string `json:"valor"`
}

func main() {
	http.HandleFunc("/cotacao", buscaDolarhandler)
	http.ListenAndServe(":8080", nil)

}

/*
==========================================================
  - Função: buscaDolarhandler
  - Descrição : Função executada ao se acessar o endpoint
  - /cotacao.
  - Parametros :
  - res - Resposta do tipo: http.ResponseWriter
  - req - Ponteiro de Requisição do tipo: http.Request
  - Retorno:

==========================================================
*/
func buscaDolarhandler(res http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	//Aplica conf de timeout de 200ms
	ctx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)

	defer cancel() //Cancela Contexto

	log.Println("Request Iniciada.")
	defer log.Println("Request finalizada.")

	//Cria request para https://economia.awesomeapi.com.br/json/last/USD-BRL
	req, err := http.NewRequestWithContext(ctx, "GET", "https://economia.awesomeapi.com.br/json/last/USD-BRL", nil)
	if err != nil {
		log.Println("Erro ao criar request: ", err)
		return //panic(err)
	}

	//Aplica resquest
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("Erro ao aplicar a request: ", err)
		return //panic(err)
	}
	defer resp.Body.Close() //Fecha o body

	//Lê o body da resposta
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("Erro ao ler o body da resposta: ", err)
		http.Error(res, "Erro ao ler o body da resposta.", http.StatusInternalServerError)
		return //panic(err)
	}

	// Struct de retorno da request
	var dados map[string]struct {
		Bid string `json:"bid"`
	}

	err = json.Unmarshal(body, &dados)
	if err != nil {
		log.Println("Erro ao fazer parsing da resposta: ", err)
		http.Error(res, "Erro ao fazer parsing da resposta.", http.StatusInternalServerError)
		return
	}

	//Grava dados na base de dados.
	err = gravaDados(dados["USDBRL"].Bid)

	if err != nil {
		log.Println("Erro ao gravar a resposta no BD: ", err)
		http.Error(res, "Erro ao gravar a resposta no BD.", http.StatusInternalServerError) //Imprime no Browser.
		return
	}

	//Variavel que recebe o valor da cotacao (Bid)
	cotacao := dados["USDBRL"].Bid

	//Cria haeder para devolução de JSON
	res.Header().Set("Content-Type", "Application/json")
	res.WriteHeader(http.StatusOK)

	//Cria novo encoder para converter a struct em JSON e já escreve resultado no Response
	json.NewEncoder(res).Encode(map[string]string{"Dolar": cotacao})

	//Aguarda resultado do contexto e dá mensagens de sucesso ou não
	select {
	case <-ctx.Done():
		log.Println("Request Cancelada por timeout.")                                //Imprime no servidor na linha de comando.
		http.Error(res, "Request cancelada por timeout.", http.StatusRequestTimeout) //Imprime no Browser.
	default:
		log.Println("Request processada com sucesso.") //Imprime no servidor na linha de comando.
		http.ResponseWriter.Write(res, []byte("Request processada com sucesso."))
	}
}

/*
==========================================================
  - Função: gravaDados
  - Descrição : Função para gravar no banco de dados o
  - valor da cotacao.
  - Parametros :
  - dolar - Valor do dolar vindo da chamada tipo: string
  - Retorno: Erro se houver problema ou nulo caso contrá-
  - rio.

==========================================================
*/
func gravaDados(dolar string) error {
	log.Println("gravaDados")
	//Cria contexto com timeout de 10ms
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	//sql.Open("mysql","[usr]:[pwd]@tcp(localhost:[porta do docker])/[nome db]"
	db, err := sql.Open("sqlite3", "./Cotacao.db")

	if err != nil {
		log.Println("Erro ao abrir Banco de dados: ", err)
		return err
	}
	defer db.Close()

	//Cria o comando (statement) CREATE a ser executado no DB
	stmt, err := db.Prepare("CREATE TABLE IF NOT EXISTS dolar (Id INTEGER PRIMARY KEY, Valor REAL(4,4))")
	if err != nil {
		log.Println("Erro ao criar comando SQL CREATE: ", err)
		return err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx)
	if err != nil {
		log.Println("Ocorreu um erro ao tentar criar a tabela dolar: ", err)
		return err
	}

	//Converte o string para float para inserir na base de dados
	dolarFloat, err := strconv.ParseFloat(dolar, 32)
	if err != nil {
		log.Println("Erro ao converter valor: ", err)
		return err
	}

	//Cria o comando (statement) INSERT a ser executado no DB
	stmt, err = db.Prepare("insert into dolar (Valor) values (?)")
	if err != nil {
		log.Println("Erro ao criar comando INSERT em dolar: ", err)
		return err
	}

	//Executa o comando no DB.
	_, err = stmt.ExecContext(ctx, dolarFloat)
	if err != nil {
		log.Println("Erro ao executar comando SQL: ", err)
		return err
	}

	select {
	case <-ctx.Done():
		log.Println("Request Cancelada por timeout.") //Imprime no servidor na linha de comando.
		return ctx.Err()
	default:
		log.Println("Request processada com sucesso.") //Imprime no servidor na linha de comando.
		//res.Write([]byte("Resquest processada com sucesso.")) //Imprime no Browser.
	}
	return nil
}
