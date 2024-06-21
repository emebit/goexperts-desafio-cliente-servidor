/*
=====================================================================================================*\
  - Client.go : Deverá realizar uma requisição HTTP no server.go solicitando a cotação do dólar.
  - precisará receber do server.go apenas o valor atual do câmbio (campo "bid" do JSON).
  - Utilizando o package "context", terá um timeout máximo de 300ms para receber o
  - resultado do server.go.
  - Terá que salvar a cotação atual em um arquivo "cotacao.txt" no formato: Dólar: {valor}.

\*=====================================================================================================
*/
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// Estrutura para devolver JSON
type Cotacao struct {
	Dolar string `json:"dolar"`
}

func main() {
	//Cria contexto com timeout de 300ms
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	//Cria requisição para server
	req, err := http.NewRequestWithContext(ctx, "GET", "http://localhost:8080/cotacao", nil)
	if err != nil {
		log.Println("Erro ao criar requisição para servidor: ", err)
		return //panic(err)
	}

	//Executa a requisição
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("Erro ao executar a requisição: ", err)
		return //panic(err)
	}
	defer res.Body.Close() //Fecha o Body

	//Lê o body da resposta
	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Println("Erro ao ler o body da resposta: ", err)
		return //panic(err)
	}

	//Estrutura para receber o conteudo do JSON
	var cotacao Cotacao

	err = json.Unmarshal(body, &cotacao)
	if err != nil {
		log.Println("Erro ao fazer parsing da resposta: ", err)
		return
	}

	//Grava dados no arquivo.
	err = gravaArquivo("cotacao.txt", cotacao.Dolar)
	if err != nil {
		log.Println("Erro ao gravar a resposta no arquivo: ", err)
		return
	}

	//Aguarda resultado do contexto e dá mensagens de sucesso ou não
	select {
	case <-ctx.Done():
		log.Println("Request Cancelada por timeout.") //Imprime no servidor na linha de comando.
	default:
		log.Println("Request processada com sucesso.") //Imprime no servidor na linha de comando.
	}
}

/*
=============================================================
  - Função: gravaArquivo
  - Descrição : Função que grava o valor da cotacao em um
  - arquivo texto.
  - Parametros :
  - nomeArq - Nome do arquivo do tipo: string
  - valor - Valor do dolar do tipo: string
  - Retorno: Erro em caso de problema ou nulo caso contrario

=============================================================
*/
func gravaArquivo(nomeArq string, dado string) error {
	//Se arquivo não existe, cria arquivo para guardar a resposta
	arq, err := os.OpenFile(nomeArq, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return err
	}

	//Grava resposta no arquivo
	_, err = arq.WriteString(fmt.Sprintf("Dolar: %s\n", dado))
	if err != nil {
		return err
	}
	arq.Close()
	return nil
}
