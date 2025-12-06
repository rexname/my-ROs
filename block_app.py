#!/usr/bin/env python3
import re
import time
import subprocess
from socket import AF_INET
import nfqueue
from scapy.all import IP, TCP, Raw

APPS = {
    "youtube": [r".*\.youtube\.com$", r".*\.googlevideo\.com$"],
    "tiktok": [r".*\.tiktokcdn\.com$", r".*\.tiktokv\.com$"],
    "spotify": [r".*\.spotify\.com$", r".*\.scdn\.co$"],
}
REGEX = {k: [re.compile(p, re.I) for p in v] for k, v in APPS.items()}

def match_app(host):
    for app, pats in REGEX.items():
        for rx in pats:
            if rx.match(host):
                return app
    return None

def extract_host_http(data):
    try:
        s = data.decode("latin1", errors="ignore")
        i = s.find("\r\nHost:")
        if i == -1:
            i = s.find("\nHost:")
        if i == -1:
            return None
        line = s[i:].splitlines()[0]
        host = line.split(":", 1)[1].strip().split(" ")[0]
        return host
    except:
        return None

def extract_sni(data):
    try:
        if len(data) < 5 or data[0] != 22:
            return None
        pos = 5
        if data[pos] != 1:
            return None
        pos += 1
        pos += 3
        pos += 2
        pos += 32
        sid_len = data[pos]
        pos += 1 + sid_len
        cs_len = (data[pos] << 8) | data[pos + 1]
        pos += 2 + cs_len
        cm_len = data[pos]
        pos += 1 + cm_len
        ext_len = (data[pos] << 8) | data[pos + 1]
        pos += 2
        end = pos + ext_len
        while pos + 4 <= end:
            etype = (data[pos] << 8) | data[pos + 1]
            elen = (data[pos + 2] << 8) | data[pos + 3]
            pos += 4
            if etype == 0 and elen >= 5:
                list_len = (data[pos] << 8) | data[pos + 1]
                pos += 2
                if pos + 3 > end:
                    break
                pos += 1
                name_len = (data[pos] << 8) | data[pos + 1]
                pos += 2
                if pos + name_len > end:
                    break
                return data[pos:pos + name_len].decode("ascii", errors="ignore")
            pos += elen
        return None
    except:
        return None

def add_block(app, ip, port):
    setname = f"block_{app}"
    elem = f"{ip} . {port} timeout 300s"
    cmd = ["nft", "add", "element", "inet", "myapp", setname, "{ " + elem + " }"]
    subprocess.run(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE)

def cb(payload):
    raw = payload.get_data()
    try:
        pkt = IP(raw)
        if TCP not in pkt:
            payload.set_verdict(nfqueue.NF_ACCEPT)
            return
        dport = int(pkt[TCP].dport)
        dst = pkt.dst
        data = bytes(pkt[TCP].payload) if Raw in pkt[TCP] else b""
        host = None
        if dport == 443 and data:
            host = extract_sni(data)
        elif dport == 80 and data:
            host = extract_host_http(data)
        app = match_app(host) if host else None
        if app:
            add_block(app, dst, dport)
            print(f"{time.strftime('%Y-%m-%dT%H:%M:%S')} {app} {host} {dst}:{dport}", flush=True)
    except Exception as e:
        print(f"err {e}", flush=True)
    payload.set_verdict(nfqueue.NF_ACCEPT)

def main():
    q = nfqueue.queue()
    q.set_callback(cb)
    q.open()
    q.bind(AF_INET)
    q.create_queue(0)
    try:
        q.try_run()
    except KeyboardInterrupt:
        pass
    q.unbind(AF_INET)
    q.close()

if __name__ == "__main__":
    main()
