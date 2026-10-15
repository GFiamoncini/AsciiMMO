#!/usr/bin/env bash
# Inicia, para e acompanha o servidor do AsciiMMO no Linux.
#
#   ./servidor.sh iniciar     sobe em segundo plano (log em ~/AsciiMMO/servidor.log)
#   ./servidor.sh parar       para gravando todos os personagens
#   ./servidor.sh reiniciar   para e sobe de novo (use depois de compilar)
#   ./servidor.sh estado      diz se está rodando, desde quando e em que portas
#   ./servidor.sh log         acompanha o log (Ctrl+C sai só do log)
#   ./servidor.sh compilar    compila servidor e cliente (Go 1.27 baixado se preciso)
#
# Opções extras do servidor vão depois do comando, por exemplo:
#   ./servidor.sh iniciar -weather rain
set -u
cd "$(dirname "$(readlink -f "$0")")"
DADOS="${ASCIIMMO_DADOS:-$HOME/AsciiMMO}"
mkdir -p "$DADOS"
PIDF="$DADOS/servidor.pid"
LOG="$DADOS/servidor.log"
BIN="$PWD/asciimmo"

pid_atual() {
	if [ -f "$PIDF" ] && kill -0 "$(cat "$PIDF")" 2>/dev/null; then
		cat "$PIDF"
		return
	fi
	# servidor aberto à mão (sem o arquivo de pid): procura pelo binário daqui
	for p in $(pgrep -x asciimmo); do
		exe="$(readlink /proc/$p/exe 2>/dev/null)"
		if [ "${exe% (deleted)}" = "$(readlink -f "$BIN")" ]; then
			echo "$p"
			return
		fi
	done
}

compilar() {
	echo "Compilando servidor e cliente..."
	export GOTOOLCHAIN="${GOTOOLCHAIN:-go1.27.0+auto}"
	go build -buildvcs=false -o asciimmo ./servidor && go build -buildvcs=false -o asciimmo-cliente ./cliente && echo "Pronto."
}

iniciar() {
	local p
	p="$(pid_atual)"
	if [ -n "$p" ]; then
		echo "O servidor já está rodando (pid $p). Use: $0 reiniciar"
		return 1
	fi
	[ -x "$BIN" ] || compilar || return 1
	echo "--- $(date '+%d/%m/%Y %H:%M:%S') iniciando ---" >>"$LOG"
	nohup "$BIN" "$@" >>"$LOG" 2>&1 &
	echo $! >"$PIDF"
	sleep 2
	if kill -0 "$(cat "$PIDF")" 2>/dev/null; then
		echo "Servidor no ar (pid $(cat "$PIDF")). Jogo: ./asciimmo-cliente  ·  log: $0 log"
	else
		echo "O servidor não subiu. Últimas linhas do log:"
		tail -n 15 "$LOG"
		rm -f "$PIDF"
		return 1
	fi
}

parar() {
	local p
	p="$(pid_atual)"
	if [ -z "$p" ]; then
		echo "O servidor não está rodando."
		rm -f "$PIDF"
		return 0
	fi
	echo "Parando o servidor (pid $p) e gravando os personagens..."
	kill -TERM "$p"
	for _ in $(seq 1 30); do
		kill -0 "$p" 2>/dev/null || break
		sleep 0.5
	done
	if kill -0 "$p" 2>/dev/null; then
		echo "Não parou em 15 s: forçando."
		kill -9 "$p"
	fi
	rm -f "$PIDF"
	echo "Parado."
}

estado() {
	local p
	p="$(pid_atual)"
	if [ -z "$p" ]; then
		echo "Parado."
		return 1
	fi
	echo "Rodando: pid $p, desde $(ps -o lstart= -p "$p")"
	if readlink "/proc/$p/exe" 2>/dev/null | grep -q '(deleted)$'; then
		echo "Atenção: o asciimmo foi recompilado depois que este servidor subiu. Rode: $0 reiniciar"
	else
		echo "Rodando a versão compilada em $(date -r "$BIN" '+%d/%m %H:%M')."
	fi
	ss -ltnp 2>/dev/null | grep "pid=$p," | awk '{print "  porta " $4}'
}

cmd="${1:-estado}"
[ $# -gt 0 ] && shift
case "$cmd" in
iniciar | start) iniciar "$@" ;;
parar | stop) parar ;;
reiniciar | restart) parar && iniciar "$@" ;;
estado | status) estado ;;
log) tail -n 40 -f "$LOG" ;;
compilar | build) compilar ;;
*)
	sed -n '2,13p' "$0" | sed 's/^# \{0,1\}//'
	exit 1
	;;
esac
