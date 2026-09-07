Example to follow:

```go
package main

import (
	"fmt"
	"regexp"
	"sync"
	"time"
)

// CronService define a interface que todos os serviços devem implementar
type CronService interface {
	Run(Cron) error
}

type CronOption struct {
	Metadata map[string]any
}

type Cron struct {
	Interval time.Duration
	Name     string
	Service  CronService
	Option   CronOption
}

// CronManager gerencia as execuções
type CronManager struct {
	services []Cron
	quit     chan struct{}   // Usamos struct{} pois não gasta memória, serve apenas como sinal
	wg       *sync.WaitGroup // Ajuda a esperar os serviços pararem graciosamente no Stop()
}

// NewCron cria uma nova instância do gerenciador
func NewCron() *CronManager {
	return &CronManager{
		services: make([]Cron, 0),
		quit:     make(chan struct{}),
		wg:       &sync.WaitGroup{},
	}
}

// Register adiciona uma nova rotina ao gerenciador
func (c *CronManager) Register(cron Cron) {
	rgx := regexp.MustCompile(`^[a-z-]+$`)

	if !rgx.MatchString(cron.Name) {
		panic(fmt.Sprintf("Nome de cron inválido: '%s'. Use apenas letras minúsculas e hífen.", cron.Name))
	}

	c.services = append(c.services, cron)
}

// runInterval é a engrenagem central (o verdadeiro setInterval)
func (c *CronManager) runInterval(cron Cron) {
	defer c.wg.Done() // Avisa que esta goroutine terminou quando a função retornar

	// O Ticker é o equivalente perfeito ao setInterval
	ticker := time.NewTicker(cron.Interval)
	defer ticker.Stop() // Evita vazamento de memória

	for {
		// O select espera até que algum dos canais receba uma mensagem
		select {
		case <-ticker.C:
			// Disparado toda vez que o intervalo passa
			err := cron.Service.Run(cron)
			if err != nil {
				fmt.Printf("[Erro] Falha no serviço %s: %v\n", cron.Name, err)
			}
		case <-c.quit:
			// Disparado quando fechamos o canal quit no método Stop()
			fmt.Printf("[Parando] Serviço: %s\n", cron.Name)
			return
		}
	}
}

// Start inicia todos os serviços registrados em background
func (c *CronManager) Start() {
	for _, cron := range c.services {
		c.wg.Add(1) // Adiciona 1 ao contador de goroutines ativas
		go c.runInterval(cron)
		fmt.Printf("[Iniciado] Serviço: %s (Intervalo: %s)\n", cron.Name, cron.Interval)
	}
}

// Stop para todos os serviços e espera que finalizem
func (c *CronManager) Stop() {
	fmt.Println("\nSinal de parada enviado. Aguardando serviços finalizarem...")
	close(c.quit) // Fechar um canal envia um sinal para todos os `select` que estão ouvindo ele
	c.wg.Wait()   // Bloqueia até que todos os `defer c.wg.Done()` sejam chamados
	fmt.Println("Todos os serviços foram parados com sucesso.")
}

// ==========================================
// EXEMPLO DE USO
// ==========================================

// MeuServico de exemplo que implementa a interface CronService
type MeuServico struct {
	Mensagem string
}

func (s *MeuServico) Run(c Cron) error {
	fmt.Printf("[%s] Executando... %s\n", time.Now().Format("15:04:05"), s.Mensagem)
	return nil
}

func main() {
	manager := NewCron()

	// Registrando serviço 1 (a cada 2 segundos)
	manager.Register(Cron{
		Name:     "ping-service",
		Interval: 2 * time.Second,
		Service:  &MeuServico{Mensagem: "Ping!"},
	})

	// Registrando serviço 2 (a cada 3 segundos)
	manager.Register(Cron{
		Name:     "limpeza-cache",
		Interval: 3 * time.Second,
		Service:  &MeuServico{Mensagem: "Limpando cache da aplicação..."},
	})

	// Inicia os serviços em goroutines independentes
	manager.Start()

	// Mantém o programa principal rodando por 10 segundos
	time.Sleep(10 * time.Second)

	// Para a "Cron" graciosamente
	manager.Stop()
}
```