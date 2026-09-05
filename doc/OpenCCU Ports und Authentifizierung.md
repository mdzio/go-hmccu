# OpenCCU: Ports und Authentifizierung

## Übersicht

OpenCCU nutzt **lighttpd** als vorgeschalteten Reverse Proxy. Die Backend-Dienste
lauschen nur auf `127.0.0.1` auf internen Ports. Nach außen hin stellt lighttpd
Legacy-Ports (Klartext) und jeweils TLS-Varianten im 4xxxx-Bereich bereit, die auf
die internen Ports weiterleiten.

---

## 1. Ports

### 1.1 Web-UI (lighttpd -> ReGaHss / HMServer)

| Extern (lighttpd) | Intern (Backend) | Dienst | Quelle |
|---|---|---|---|
| 80 (HTTP) | 8183 | ReGaHss Script (Web-UI) | `proxy.conf`, `webui.conf`, `rega.conf:45` |
| 443 (HTTPS/TLS) | 8183 | ReGaHss Script (Web-UI) | `sslsettings.conf:1`, `proxy.conf:27` |
| 80 / 443 | 39292 | HMServer (`/pages/jpages`) | `webui.conf:13` |

Die Web-UI wird primär über **80/443** ausgeliefert; lighttpd leitet an den
ReGaHss-Script-Port **8183** weiter. Statische/Java-Seiten unter `/pages/jpages`
gehen an den HMServer (39292).

### 1.2 Funkschnittstellen (XmlRpc der RF-Dienste)

| Extern (Klartext) | Extern (TLS) | Intern | Dienst | Quelle |
|---|---|---|---|---|
| 2001 | 42001 | 32001 | RFD (BidCos-RF) | `rfd.port`, `webui_remoteapi.conf:118` |
| 2000 | 42000 | 32000 | HS485D (HomeMatic Wired) | `hs485d.port`, `webui_remoteapi.conf:142` |
| 9292 | 49292 | 39292 | HMServer (HmIP) | `HMServer.conf:1`, `webui_remoteapi.conf:94` |
| 2010 | 42010 | 32010 | crRFD Legacy (HmIP) | `webui_remoteapi.conf:70` |

Hinweis: `migrateSecuritySettings.sh` zeigt, dass RFD/HS485D frueher direkt auf
2001/2000 lauschten und mittlerweile auf die internen 32xxx-Ports umgezogen sind;
lighttpd haelt die alten Ports als Kompatibilitaets-Eingaenge vor.

### 1.3 ReGaHss (Logikschicht)

| Extern (Klartext) | Extern (TLS) | Intern | Funktion | Quelle |
|---|---|---|---|---|
| 8181 | 48181 | 8183 | ReGaHss Script (HTTP-UI-Backend) | `rega.conf:45` (`HttpListenerPort`), `webui_remoteapi.conf:2` |
| 1999 | 41999 | 31999 | ReGaHss XmlRpc | `rega.conf:98` (`XmlRpcServerPort`), `HMServer.conf:2` (`regaPort`), `webui_remoteapi.conf:46` |

### 1.4 Zusammenfassung der Kern-Ports

- **Web-UI nach außen:** 80 (HTTP) und 443 (HTTPS) -- intern an ReGaHss 8183 weitergeleitet.
- **Funkschnittstellen nach außen:** 2001/42001 (RFD), 2000/42000 (HS485D), 9292/49292 (HMServer/HmIP), 2010/42010 (crRFD) -- intern 32001, 32000, 39292, 32010.
- **ReGaHss nach außen:** 8181/48181 (Script) und 1999/41999 (XmlRpc) -- intern 8183 bzw. 31999.

Alle externen Ports werden über `webui_remoteapi.conf` (bzw.
`webui_remoteapi_notready.conf` im Nicht-bereit-Zustand) von lighttpd gebunden
und pro Port auf `127.0.0.1` an den jeweiligen Backend-Dienst weitergereicht.
Die TLS-Varianten folgen dem Schema `4` + Legacy-Port (z. B. 2001 -> 42001,
9292 -> 49292).

### 1.5 Firewall-Service-Definitionen

In `libfirewall.tcl:27-30` sind die Service-Ports fuer die iptables-basierte
Firewall definiert:

```
XMLRPC:    2000 2001 2002 2010 9292 42000 42001 42010 49292
REGA:      8181 1999 48181 41999
NEOSERVER: 8088 9099 1901 1902 5987 10000 48899 49880
SNMP:      161
```

Die Zugriffsstufen `full` / `restricted` / `none` steuern, ob diese Ports
ueberhaupt erreichbar sind.

---

## 2. Authentifizierung

OpenCCU authentifiziert an zwei Stellen: auf **HTTP-Ebene ueber lighttpd**
(HTTP Basic Auth gegen ReGaHss) und auf **Netzwerkebene ueber iptables**
(Firewall/Security-Level).

### 2.1 HTTP Basic Authentication ueber lighttpd (`mod_authn_rega`)

Die zentrale Konfiguration steht in
`overlay/WebUI/etc/lighttpd/conf.d/auth.conf`:

```
$HTTP["remoteip"] !~ "^(?:127\.0\.0\.1|::ffff:127\.0\.0\.1|::1)$" {
  auth.backend = "rega"
  auth.backend.rega.port = 1998
  auth.cache = ("max-age" => "600")

  auth.require = ( "/" =>
    (
      "method" => "basic",
      "realm" => "theRealm",
      "require" => "user=user"
    )
  )
}
```

| Eigenschaft | Wert | Bedeutung |
|---|---|---|
| Backend | `rega` | Custom lighttpd-Modul `mod_authn_rega` (geladen in `modules.conf:41`) |
| Backend-Port | 1998 | ReGaHss-Authentifizierungsdienst (UDP), aktiviert via `rega.conf:108` `AuthenticatorServiceEnabled=true` |
| Methode | `basic` | HTTP Basic Authentication |
| Realm | `theRealm` | -- |
| require | `user=user` | Angemeldeter Benutzer erforderlich |
| Cache | 600 s | Auth-Ergebnisse werden 10 Minuten gecacht |
| Geltungsbereich | alle Remote-IPs außer localhost | Nur externe Zugriffe werden authentifiziert; `127.0.0.1`/`::1` sind freigestellt |

Die Authentifizierung greift fuer **alle externen Zugriffe** auf die ueber
lighttpd ausgelieferten Ports (80, 443 und die Remote-API-Ports 8181/48181,
1999/41999, 2000/42000, 2001/42001, 2010/42010, 9292/49292), sofern die
Auth-Konfiguration aktiv ist.

### 2.2 Aktivierung/Deaktivierung der Authentifizierung

Die Authentifizierung ist nicht immer aktiv -- sie wird ueber eine Flag-Datei
gesteuert. Das Init-Skript `overlay/WebUI/etc/init.d/S50lighttpd` generiert zur
Laufzeit die Datei `/var/etc/lighttpd_auth.conf`:

- Wenn `/etc/config/authEnabled` existiert **und** `/etc/lighttpd/conf.d/auth.conf`
  vorhanden ist:
  ```
  include "/etc/lighttpd/conf.d/auth.conf"
  ```
  -> Authentifizierung aktiv.
- Sonst:
  ```
  (leer)
  ```
  -> Keine Authentifizierung.

Diese Logik laeuft bei `start()` und `reload()` (Zeilen 42-46 und 90-94). Ein
`reload` von lighttpd reicht also, um die Auth zu aktivieren/deaktivieren.

### 2.3 Steuerung ueber Security-Level

Die Datei `/etc/config/authEnabled` wird durch `libsecuritylevel.tcl` gesetzt
(`SEC_setAuthEnabled`):

| Security-Level | XMLRPC | REGA | Auth aktiv | Quelle |
|---|---|---|---|---|
| LOW | full | restricted | nein (`SEC_setAuthEnabled false`) | `libsecuritylevel.tcl:27` |
| MEDIUM | restricted | restricted | ja (`SEC_setAuthEnabled true`) | `libsecuritylevel.tcl:38` |
| HIGH | none | none | ja (`SEC_setAuthEnabled true`) | `libsecuritylevel.tcl:49` |

`SEC_setAuthEnabled true` erzeugt `/etc/config/authEnabled` per `touch`; `false`
entfernt sie per `rm -f` (`libsecuritylevel.tcl:116-122`).

### 2.4 Firewall als zusaetzliche Zugriffskontrolle (Netzwerkebene)

Parallel zur HTTP-Auth kontrolliert `libfirewall.tcl` den Zugriff auf die
Service-Ports ueber iptables. Die Zugriffsstufen `full` / `restricted` / `none`
steuern, ob diese Ports ueberhaupt erreichbar sind -- unabhaengig von der
HTTP-Auth. Bei `none` blockiert die Firewall den Port komplett; bei `restricted`
wird der Zugriff auf lokale Netze begrenzt.

### 2.5 Ausnahmen in der Web-UI-Konfiguration

In `webui.conf:1` wird `auth.require = ()` gesetzt, was die Auth-Anforderung
fuer die regulaere Web-UI (Port 80/443) explizit deaktiviert. Die
Authentifizierung der Web-UI selbst erfolgt stattdessen ueber die
ReGaHss-Session-Logik (Login-Formular), nicht ueber HTTP Basic Auth. Basic Auth
greift primaer auf die Remote-API-Ports (XmlRpc/Script-Schnittstellen).

---

## 3. Port 1998: ReGaHss Authenticator-Service

Port 1998 ist kein eigenstaendiger Dienst, sondern ein integraler Bestandteil
des **ReGaHss**-Prozesses. ReGaHss startet mit `/etc/rega.conf`
(`S70ReGaHss:113`), und dort ist der Authenticator-Service aktiviert:

```
# rega.conf:104-108
AuthenticatorServiceEnabled=true
```

ReGaHss oeffnet bei Aktivierung dieses Flags einen **UDP-Socket auf Port 1998**
(nur auf `127.0.0.1`/Loopback) und beantwortet Authentifizierungsanfragen.

### 3.1 Protokoll: Wie die Authentifizierung funktioniert

Das lighttpd-Modul `mod_authn_rega` (Quelle:
`patches/lighttpd/0001-rega-auth/lighttpd/src/mod_authn_rega.c`) implementiert
den Client-Teil. Der Ablauf bei einer HTTP-Basic-Auth-Anfrage:

```
Client (Browser/API)
  |  HTTP Basic Auth (Base64: user:password)
  v
lighttpd (mod_auth + mod_authn_rega)
  |  UDP-Paket an 127.0.0.1:1998
  |  Nachricht: "<user>:<pass>"
  |  (Backslash und Doppelpunkt werden escaped)
  v
ReGaHss (AuthenticatorService auf :1998)
  |  prueft User/Passwort gegen interne Benutzerverwaltung
  |  Antwort: "1" = OK, alles andere = falsch
  v
lighttpd
  |  "1" -> Request durchlassen (HANDLER_GO_ON)
  |  sonst -> 401 Unauthorized (HANDLER_ERROR)
```

### 3.2 Details aus dem Code (`mod_authn_rega.c`)

| Aspekt | Wert | Quelle (Zeile) |
|---|---|---|
| Socket-Typ | `SOCK_DGRAM` / `IPPROTO_UDP` | 190 |
| Ziel-Adresse | `127.0.0.1` (Loopback) | 210 |
| Ziel-Port | konfiguriert via `auth.backend.rega.port` (1998) | 211 |
| Timeout | 1 Sekunde (send + recv) | 198-199 |
| Nachrichtenformat | `user:pass` mit Escaping von `\` und `:` | 133-171 |
| Antwort | `"1"` = authentifiziert, sonst nicht | 178 |
| Antwort-Verifikation | Prueft, dass Antwort von Port 1998 + Loopback kommt | 242-248 |

### 3.3 Ueberwachung durch Monit

Der ReGaHss-Prozess (und damit Port 1998) wird von monit ueberwacht
(`overlay/base-openccu/etc/monitrc:329`):

```
check process ReGaHss with pidfile /var/run/ReGaHss.pid
    if failed port 1998 type udp for 20 cycles then restart
```

Faellt der UDP-Service auf 1998 aus, startet monit den ReGaHss neu.

### 3.4 Zusammenfassung

Port 1998 ist ein **reiner Loopback-UDP-Port**, auf dem der ReGaHss-Prozess
einen Authenticator-Service bereitstellt. Er ist nicht von außen erreichbar und
nicht in der Firewall-Konfiguration (`libfirewall.tcl`) gelistet. lighttpd's
`mod_authn_rega` sendet bei HTTP-Basic-Auth-Anfragen die Credentials als
UDP-Paket an `127.0.0.1:1998` und erhaelt von ReGaHss eine `1` (erfolgreich)
oder eine andere Antwort (fehlgeschlagen) zurueck.

---

## 4. Quell-Dateien aus dem OpenCCU-Repository

| Datei | Funktion |
|---|---|
| `overlay/base/etc/rega.conf` | ReGaHss-Konfiguration (HttpListenerPort 8183, XmlRpcServerPort 31999, AuthenticatorServiceEnabled) |
| `overlay/base/etc/HMServer.conf` | HMServer-Ports (hmServerPort 39292, regaPort 31999) |
| `overlay/base/etc/rfd.port` | RFD-Port (32001) |
| `overlay/base/etc/hs485d.port` | HS485D-Port (32000) |
| `overlay/base/etc/lighttpd/lighttpd.conf` | lighttpd-Hauptkonfiguration |
| `overlay/base/etc/lighttpd/conf.d/sslsettings.conf` | server.port = 443, SSL-Einstellungen |
| `overlay/base/etc/lighttpd/conf.d/proxy.conf` | Port 80/443-Sockets, inkludiert auth/httpsredirect/remoteapi |
| `overlay/base/etc/lighttpd/conf.d/webui.conf` | Proxy-Regeln fuer Web-UI (-> 8183, 39292) |
| `overlay/base/etc/lighttpd/conf.d/webui_remoteapi.conf` | Proxy-Regeln fuer Remote-API-Ports (aktiv) |
| `overlay/base/etc/lighttpd/conf.d/webui_remoteapi_notready.conf` | Remote-API-Ports im Nicht-bereit-Zustand (-> notready.cgi) |
| `overlay/base/etc/lighttpd/conf.d/httpsredirect.conf` | HTTP->HTTPS-Weiterleitung |
| `overlay/base/etc/lighttpd/modules.conf` | Modul-Liste (mod_authn_rega, mod_auth, mod_proxy, ...) |
| `overlay/WebUI/etc/lighttpd/conf.d/auth.conf` | HTTP Basic Auth-Konfiguration (Backend rega, Port 1998) |
| `overlay/WebUI/etc/init.d/S50lighttpd` | lighttpd-Init (generiert /var/etc/lighttpd_*.conf) |
| `overlay/WebUI/etc/init.d/S70ReGaHss` | ReGaHss-Init |
| `overlay/base/lib/libfirewall.tcl` | Firewall-Service-Definitionen (Ports, Access-Level) |
| `overlay/base/lib/libsecuritylevel.tcl` | Security-Level-Logik (authEnabled, XMLRPC/REGA/NEOSERVER Access) |
| `overlay/base/bin/enforcesecuritylevel.tcl` | Erzwingt Security-Level beim Boot |
| `overlay/base/bin/migrateSecuritySettings.sh` | Migration alter Listen-Ports (2001->32001, 2000->32000) |
| `patches/lighttpd/0001-rega-auth/lighttpd/src/mod_authn_rega.c` | lighttpd-Modul: UDP-Auth gegen ReGaHss auf Port 1998 |
| `overlay/base-openccu/etc/monitrc` | Monit-Ueberwachung (ReGaHss Port 8183, 31999, 1998) |
