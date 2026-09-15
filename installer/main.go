package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type Server struct {
	root  string
	token string
	mu    bool
}

type InstallRequest struct {
	AdminEmail, AdminPassword, AdminName string
	AppPort, PostgresPort, RedisPort     string
}
type Check struct {
	Docker  bool   `json:"docker"`
	Compose bool   `json:"compose"`
	Source  bool   `json:"source"`
	Message string `json:"message"`
}

type Result struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
	URL     string `json:"url,omitempty"`
}

func main() {
	rootFlag := flag.String("root", "", "NusaMedia source root")
	addr := flag.String("addr", "127.0.0.1:18765", "installer listen address")
	flag.Parse()
	root := *rootFlag
	if root == "" {
		root = inferRoot()
	}
	root, _ = filepath.Abs(root)
	token := randomSecret(32)
	s := &Server{root: root, token: token}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", s.index)
	mux.HandleFunc("GET /api/check", s.check)
	mux.HandleFunc("POST /api/install", s.install)
	mux.HandleFunc("POST /api/stop", s.stop)
	srv := &http.Server{Addr: *addr, Handler: security(mux), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 120 * time.Second, IdleTimeout: 60 * time.Second}
	fmt.Printf("NusaMedia Installer\nSource: %s\nOpen: http://%s/?token=%s\n", root, *addr, token)
	_ = openBrowser("http://" + *addr + "/?token=" + token)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func inferRoot() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	d, _ := filepath.Abs(filepath.Dir(exe))
	if filepath.Base(d) == "bin" {
		return filepath.Dir(filepath.Dir(d))
	}
	if filepath.Base(d) == "installer" {
		return filepath.Dir(d)
	}
	return d
}
func randomSecret(n int) string {
	b := make([]byte, n)
	if _, e := rand.Read(b); e != nil {
		return hex.EncodeToString([]byte(fmt.Sprint(time.Now().UnixNano())))
	}
	return hex.EncodeToString(b)
}

func security(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != "127.0.0.1:18765" && r.Host != "localhost:18765" { /* custom addr still served locally */
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) auth(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Query().Get("token") != s.token && r.Header.Get("X-Installer-Token") != s.token {
		http.Error(w, "forbidden", 403)
		return false
	}
	return true
}
func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	if !s.auth(w, r) {
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	template.Must(template.New("x").Parse(page)).Execute(w, map[string]any{"Root": s.root, "Token": s.token})
}
func (s *Server) check(w http.ResponseWriter, r *http.Request) {
	if !s.auth(w, r) {
		return
	}
	source := fileExists(filepath.Join(s.root, "docker-compose.yml")) && fileExists(filepath.Join(s.root, "apps", "api"))
	docker := runOK("docker", "version", "--format", "{{.Server.Version}}")
	compose := runOK("docker", "compose", "version")
	msg := "Siap"
	if !source {
		msg = "Folder source NusaMedia tidak lengkap"
	}
	if !docker {
		msg = "Docker Engine belum aktif/terpasang"
	}
	if docker && !compose {
		msg = "Docker Compose v2 belum tersedia"
	}
	jsonOut(w, Check{Docker: docker, Compose: compose, Source: source, Message: msg})
}

func (s *Server) install(w http.ResponseWriter, r *http.Request) {
	if !s.auth(w, r) {
		return
	}
	var in InstallRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&in); err != nil {
		jsonOut(w, Result{Message: "Input installer tidak valid"})
		return
	}
	if len(in.AdminPassword) < 12 || in.AdminEmail == "" || in.AdminName == "" {
		jsonOut(w, Result{Message: "Email, nama Admin Root, dan password minimal 12 karakter wajib diisi"})
		return
	}
	if !runOK("docker", "version", "--format", "{{.Server.Version}}") || !runOK("docker", "compose", "version") {
		jsonOut(w, Result{Message: "Docker Engine + Compose v2 wajib aktif sebelum instalasi"})
		return
	}
	appPort := validPort(in.AppPort, "8080")
	pgPort := validPort(in.PostgresPort, "5432")
	redisPort := validPort(in.RedisPort, "6379")
	pgpass := randomSecret(24)
	jwt := randomSecret(48)
	setup := randomSecret(32)
	env := fmt.Sprintf("POSTGRES_DB=nusamedia\nPOSTGRES_USER=nusamedia\nPOSTGRES_PASSWORD=%s\nJWT_SECRET=%s\nNUSAMEDIA_SETUP_TOKEN=%s\nAPP_PORT=%s\nPOSTGRES_PORT=%s\nREDIS_PORT=%s\nALLOWED_ORIGINS=http://localhost:%s\nALLOW_DEVELOPMENT_TOPUP=false\n", pgpass, jwt, setup, appPort, pgPort, redisPort, appPort)
	if err := os.WriteFile(filepath.Join(s.root, ".env"), []byte(env), 0600); err != nil {
		jsonOut(w, Result{Message: "Gagal menulis konfigurasi .env: " + err.Error()})
		return
	}
	if out, err := run(s.root, "docker", "compose", "up", "-d", "--build"); err != nil {
		jsonOut(w, Result{Message: "Docker gagal menjalankan NusaMedia: " + trim(out)})
		return
	}
	if err := waitComposePostgres(s.root, 60*time.Second); err != nil {
		jsonOut(w, Result{Message: "PostgreSQL belum siap: " + err.Error()})
		return
	}
	for _, migration := range []string{"001_core.sql", "002_platform_settings.sql", "003_product_platform.sql", "004_control_plane.sql", "005_organization_workspace.sql", "006_workflow_engine.sql"} {
		out, err := run(s.root, "docker", "compose", "exec", "-T", "postgres", "psql", "-U", "nusamedia", "-d", "nusamedia", "-v", "ON_ERROR_STOP=1", "-f", "/docker-entrypoint-initdb.d/"+migration)
		if err != nil {
			jsonOut(w, Result{Message: "Migrasi " + migration + " gagal: " + trim(out)})
			return
		}
	}
	api := fmt.Sprintf("http://127.0.0.1:%s", appPort)
	if err := waitHTTP(api+"/api/health", 45*time.Second); err != nil {
		jsonOut(w, Result{Message: "API belum sehat: " + err.Error()})
		return
	}
	payload, _ := json.Marshal(map[string]string{"email": in.AdminEmail, "password": in.AdminPassword, "display_name": in.AdminName})
	req, _ := http.NewRequest("POST", api+"/api/v1/setup/bootstrap", strings.NewReader(string(payload)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-NusaMedia-Setup-Token", setup)
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		jsonOut(w, Result{Message: "Gagal membuat Admin Root: " + err.Error()})
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode/100 != 2 {
		jsonOut(w, Result{Message: "Bootstrap Admin Root gagal: " + trim(string(body))})
		return
	}
	// Remove one-time bootstrap secret from persisted environment after successful bootstrap.
	env2 := strings.ReplaceAll(env, "NUSAMEDIA_SETUP_TOKEN="+setup+"\n", "")
	_ = os.WriteFile(filepath.Join(s.root, ".env"), []byte(env2), 0600)
	jsonOut(w, Result{OK: true, Message: "Instalasi selesai. Admin Root berhasil dibuat.", URL: api})
}

func (s *Server) stop(w http.ResponseWriter, r *http.Request) {
	if !s.auth(w, r) {
		return
	}
	if out, err := run(s.root, "docker", "compose", "down"); err != nil {
		jsonOut(w, Result{Message: trim(out)})
		return
	}
	jsonOut(w, Result{OK: true, Message: "NusaMedia dihentikan."})
}
func jsonOut(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
func fileExists(p string) bool               { _, e := os.Stat(p); return e == nil }
func runOK(name string, args ...string) bool { _, e := run("", name, args...); return e == nil }
func run(dir, name string, args ...string) (string, error) {
	c := exec.Command(name, args...)
	if dir != "" {
		c.Dir = dir
	}
	b, e := c.CombinedOutput()
	return string(b), e
}
func trim(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 900 {
		s = s[len(s)-900:]
	}
	return s
}
func validPort(v, d string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return d
	}
	for _, r := range v {
		if r < '0' || r > '9' {
			return d
		}
	}
	return v
}
func waitComposePostgres(root string, timeout time.Duration) error {
	end := time.Now().Add(timeout)
	for time.Now().Before(end) {
		if out, err := run(root, "docker", "compose", "exec", "-T", "postgres", "pg_isready", "-U", "nusamedia", "-d", "nusamedia"); err == nil && strings.Contains(out, "accepting connections") {
			return nil
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("timeout menunggu PostgreSQL")
}

func waitHTTP(url string, timeout time.Duration) error {
	end := time.Now().Add(timeout)
	for time.Now().Before(end) {
		r, e := http.Get(url)
		if e == nil {
			r.Body.Close()
			if r.StatusCode < 500 {
				return nil
			}
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("timeout menunggu %s", url)
}
func openBrowser(url string) error {
	var c *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		c = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		c = exec.Command("open", url)
	default:
		c = exec.Command("xdg-open", url)
	}
	return c.Start()
}

const page = `<!doctype html><html lang="id"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>NusaMedia Installer</title><style>*{box-sizing:border-box}body{margin:0;font-family:Inter,system-ui,sans-serif;background:#f7f7f8;color:#171717}.wrap{max-width:920px;margin:auto;padding:28px 18px}.brand{font-weight:800;font-size:25px}.card{background:white;border:1px solid #e7e7e9;border-radius:20px;padding:24px;margin-top:18px;box-shadow:0 12px 40px #0000000a}.steps{display:flex;gap:8px;overflow:auto;margin:18px 0}.step{padding:9px 12px;border-radius:999px;background:#eee;font-size:13px;white-space:nowrap}.step.on{background:#b91c1c;color:white}.grid{display:grid;grid-template-columns:1fr 1fr;gap:14px}label{font-size:13px;font-weight:700;display:block}input{width:100%;padding:12px;border:1px solid #d5d5d8;border-radius:10px;margin-top:7px;font:inherit}button{border:0;border-radius:11px;padding:12px 16px;font-weight:800;cursor:pointer}.primary{background:#b91c1c;color:#fff}.secondary{background:#eee}.row{display:flex;gap:10px;flex-wrap:wrap;margin-top:18px}.status{padding:12px;border-radius:12px;background:#f2f2f3;margin-top:12px}.ok{background:#ecfdf3;color:#166534}.bad{background:#fef2f2;color:#991b1b}.muted{color:#666}.small{font-size:12px}@media(max-width:680px){.grid{grid-template-columns:1fr}.card{padding:18px}.wrap{padding:18px 12px}}</style></head><body><main class="wrap"><div class="brand">NusaMedia <span class="muted">Installer</span></div><div class="muted">Browser-based setup wizard · local-only</div><div class="steps"><div class="step on" id="s1">1 Persiapan</div><div class="step" id="s2">2 Konfigurasi</div><div class="step" id="s3">3 Admin Root</div><div class="step" id="s4">4 Instalasi</div><div class="step" id="s5">5 Selesai</div></div><section class="card"><h2>Instal NusaMedia</h2><p class="muted">Wizard ini menjalankan NusaMedia dari source asli menggunakan Docker Compose. Password Admin Root hanya dikirim saat bootstrap dan tidak ditulis ke file konfigurasi.</p><div id="status" class="status">Memeriksa environment…</div><div id="config" hidden><h3>Konfigurasi layanan</h3><div class="grid"><label>Port Web/API<input id="app" value="8080"></label><label>Port PostgreSQL<input id="pg" value="5432"></label><label>Port Redis<input id="redis" value="6379"></label></div><h3>Admin Root</h3><div class="grid"><label>Email<input id="email" type="email" autocomplete="off"></label><label>Nama tampilan<input id="name" autocomplete="off"></label><label>Password (min. 12)<input id="pass" type="password" autocomplete="new-password"></label><label>Konfirmasi password<input id="pass2" type="password" autocomplete="new-password"></label></div><div class="row"><button class="primary" onclick="install()">Install NusaMedia</button></div></div><div id="done" hidden><div class="status ok"><b>Instalasi berhasil.</b><br>NusaMedia sudah berjalan.</div><div class="row"><button class="primary" onclick="openApp()">Buka NusaMedia</button><button class="secondary" onclick="stopApp()">Stop Services</button></div></div></section><p class="small muted">Source: {{.Root}} · Installer terikat ke localhost.</p></main><script>const token={{printf "%q" .Token}};const h={'X-Installer-Token':token};async function check(){try{let r=await fetch('/api/check?token='+encodeURIComponent(token),{headers:h});let x=await r.json();let st=document.getElementById('status');st.textContent=x.message;st.className='status '+(x.docker&&x.compose&&x.source?'ok':'bad');if(x.docker&&x.compose&&x.source){document.getElementById('config').hidden=false;document.getElementById('s2').classList.add('on')}}catch(e){document.getElementById('status').textContent='Installer tidak dapat memeriksa environment.'}}async function install(){let pass=document.getElementById('pass').value,pass2=document.getElementById('pass2').value;if(pass!==pass2){alert('Konfirmasi password tidak sama');return}document.getElementById('s4').classList.add('on');let st=document.getElementById('status');st.className='status';st.textContent='Menjalankan Docker build, migrasi database, health check, dan bootstrap Admin Root…';let body={AdminEmail:document.getElementById('email').value,AdminPassword:pass,AdminName:document.getElementById('name').value,AppPort:document.getElementById('app').value,PostgresPort:document.getElementById('pg').value,RedisPort:document.getElementById('redis').value};let r=await fetch('/api/install?token='+encodeURIComponent(token),{method:'POST',headers:{...h,'Content-Type':'application/json'},body:JSON.stringify(body)});let x=await r.json();if(x.ok){st.className='status ok';st.textContent=x.message;document.getElementById('config').hidden=true;document.getElementById('done').hidden=false;document.getElementById('s5').classList.add('on')}else{st.className='status bad';st.textContent=x.message;document.getElementById('s4').classList.remove('on')}}function openApp(){window.open('http://localhost:'+document.getElementById('app').value,'_blank')}async function stopApp(){await fetch('/api/stop?token='+encodeURIComponent(token),{method:'POST',headers:h});alert('Services dihentikan.')}check()</script></body></html>`
