# Red Team Operator — Skills Reference

---

## 1. Reconnaissance

### Passive Reconnaissance
- OSINT collection (Shodan, Censys, FOFA, Greynoise)
- Domain/subdomain enumeration (Amass, subfinder, dnsx, assetfinder)
- WHOIS, ASN, and BGP analysis
- Certificate transparency log mining (crt.sh, certspotter)
- Google/Bing dork queries for exposed assets and credentials
- LinkedIn, GitHub, Pastebin, and social media profiling
- Email harvesting (theHarvester, hunter.io)
- Metadata extraction from public documents (ExifTool, FOCA)
- Wayback Machine / archive scraping for historical exposure
- Cloud asset discovery (AWS S3 buckets, Azure blobs, GCP storage)

### Active Reconnaissance
- Network sweeping (nmap, masscan, zmap)
- DNS zone transfers and brute-force
- Web crawling and spidering (Katana, gospider, hakrawler)
- Banner grabbing and service fingerprinting
- HTTP header analysis and technology identification (Wappalyzer, WhatWeb)
- WAF/CDN detection and bypass research

---

## 2. Enumeration

### Network Enumeration
- Port scanning and service version detection (`nmap -sV -sC`)
- UDP service enumeration
- SNMP community string bruteforce (onesixtyone, snmpwalk)
- LDAP enumeration (ldapsearch, windapSearch, BloodHound ingestors)
- SMB enumeration (enum4linux-ng, CrackMapExec, Impacket)
- NFS/RPC mount enumeration
- NetBIOS and LLMNR/mDNS analysis

### Active Directory Enumeration
- BloodHound / SharpHound data collection and graph analysis
- AD user, group, GPO, and OU enumeration
- Kerberoastable and AS-REP-roastable account identification
- ACL/ACE abuse path discovery
- Domain trust mapping and forest/domain boundary analysis
- LAPS, gMSA, and privileged account identification
- SPN enumeration

### Web Application Enumeration
- Directory and file brute-forcing (feroxbuster, ffuf, dirsearch)
- Virtual host (vhost) enumeration
- API endpoint discovery and fuzzing
- Parameter enumeration (Arjun, x8)
- JavaScript file analysis for secrets and endpoints
- Identified technology version mapping to CVE databases

### Cloud Enumeration
- IAM role and permission enumeration (AWS, Azure, GCP)
- Storage bucket/blob listing and public access checks
- Serverless function discovery
- Cloud metadata service probing (169.254.169.254, IMDSv2)

---

## 3. Vulnerability Analysis

- Manual code review for logic flaws and injection points
- CVE research and PoC identification (NVD, Exploit-DB, GitHub)
- Automated scanning (Nuclei, OpenVAS, Nessus) with template customization
- Business logic flaw identification
- Authentication/authorization weakness analysis (broken access control, IDOR)
- Cryptographic weakness assessment
- Third-party/supply chain component analysis
- Configuration and hardening gap review (CIS Benchmarks)

---

## 4. Exploitation

### Web Application Exploitation
- SQL Injection (manual and automated — sqlmap, manual time-based)
- Cross-Site Scripting (reflected, stored, DOM-based)
- Server-Side Template Injection (SSTI)
- XML External Entity (XXE) injection
- Server-Side Request Forgery (SSRF)
- Command injection and OS command chaining
- Insecure deserialization exploitation
- Authentication bypass techniques (JWT manipulation, OAuth abuse)
- File upload bypasses and web shell deployment
- Path traversal and local/remote file inclusion

### Network and Service Exploitation
- Exploit framework usage (Metasploit, custom PoC scripts)
- Known CVE exploitation against unpatched services
- Password spraying and credential stuffing (Spray, MSOLSpray)
- Brute-force attacks with lockout-aware throttling
- Relay attacks (NTLM relay, NTLMv2 relay with Responder + ntlmrelayx)
- Kerberoasting and AS-REP roasting (hashcat, john)
- Pass-the-Hash (PTH) and Pass-the-Ticket (PTT)
- DCSync and secretsdump attacks
- PrintSpooler, PetitPotam, and coerce-based authentication abuse

### Client-Side Exploitation
- Phishing payload delivery (macro documents, HTML smuggling, LNK files)
- Browser exploitation via malicious links or documents
- Social engineering and pretexting (phone, email)
- Physical access scenarios (USB drops, rogue AP)

---

## 5. Post-Exploitation

### Local Privilege Escalation (Linux)
- SUID/SGID binary abuse
- Sudo misconfiguration exploitation (GTFOBins)
- Writable cron jobs and path hijacking
- Kernel exploit identification and deployment
- Docker/container escape techniques
- Capabilities abuse (cap_net_raw, cap_sys_admin)

### Local Privilege Escalation (Windows)
- Service binary path hijacking
- Unquoted service path exploitation
- DLL hijacking and DLL search-order abuse
- Scheduled task abuse
- AlwaysInstallElevated exploitation
- Token impersonation (Juicy Potato, PrintSpoofer, GodPotato)
- UAC bypass techniques

### Credential Access
- LSASS memory dumping (Mimikatz, nanodump, pypykatz)
- SAM and SYSTEM hive extraction
- DPAPI master key abuse and credential decryption
- Browser credential extraction
- Credential manager and Windows vault pillaging
- SSH key and config file harvesting
- Password file and history file hunting

### Persistence
- Registry run key and scheduled task implants
- Service installation and modification
- Boot/pre-OS persistence (bootkit concepts)
- WMI event subscriptions
- Golden Ticket and Silver Ticket creation
- Skeleton Key and Directory Services Restore Mode (DSRM) backdoors
- SSH authorized_keys modification
- Web shell persistence and timestomping

---

## 6. Lateral Movement

- Pass-the-Hash / Pass-the-Ticket across hosts
- Over-Pass-the-Hash (Overpass-the-Hash)
- Remote service exploitation (WMI, DCOM, WinRM, SCM)
- SMB/PsExec-style execution (CrackMapExec, Impacket's psexec/smbexec)
- RDP session hijacking
- SSH agent forwarding abuse
- Kerberos delegation abuse (unconstrained, constrained, resource-based constrained)
- DCOM lateral movement (MMC20.Application, ShellBrowserWindow)
- Living-off-the-land binaries (LOLBins) for execution
- AD object-based movement (WriteDACL, GenericAll, ForceChangePassword)

---

## 7. Pivoting & Tunneling

### Network Pivoting
- SSH local, remote, and dynamic port forwarding
- SOCKS5 proxy chaining (proxychains, proxychains-ng)
- Chisel (TCP/UDP tunneling over HTTP/WebSocket)
- Ligolo-ng (user-space TUN tunneling)
- Metasploit route/portfwd pivoting
- Frp (fast reverse proxy) for internal service exposure
- DNS tunneling (iodine, dnscat2)
- ICMP tunneling (ptunnel)
- HTTP/S C2 traffic tunneling through corporate proxies

### Double and Triple Pivoting
- Multi-hop SOCKS chain management
- Network segmentation mapping through successive pivots
- Maintaining pivot stability under session timeouts

---

## 8. Command & Control (C2)

- C2 framework operation (Cobalt Strike, Sliver, Havoc, Brute Ratel C4)
- Malleable C2 profile development for traffic blending
- Domain fronting and redirector configuration (Nginx, Apache, CloudFront)
- HTTPS C2 with valid certificates and CDN-based obfuscation
- DNS-based C2 channel management
- Payload staging and stageless payload deployment
- Beacon sleep, jitter, and communication interval tuning
- C2 infrastructure hardening (Mythic, categorized domains, redirectors)
- Kill-switch and self-destruct mechanisms
- Out-of-band communication channels (Slack, Teams, Discord webhooks as C2)

---

## 9. Defense Evasion

### Endpoint Evasion
- AV/EDR bypass techniques (AMSI bypass, ETW patching)
- Process injection (DLL injection, shellcode injection, process hollowing, thread hijacking)
- Reflective DLL loading
- Direct syscalls and indirect syscalls to bypass EDR hooks
- Payload encryption and obfuscation (XOR, AES, custom encoders)
- In-memory-only execution (fileless malware patterns)
- Signed binary proxy execution (LOLBins, regsvr32, mshta, certutil)
- Timestomping and artifact cleanup
- Disabling/bypassing Windows Defender and security tools

### Network Evasion
- Traffic blending with legitimate protocols (HTTPS, DNS, SMB)
- Slow and low scanning to avoid detection thresholds
- Source IP rotation and use of residential proxies
- Beacon traffic timing randomization
- Domain categorization and aging for trust building
- SSL/TLS certificate mimicry

### Log Evasion
- Windows Event Log clearing and manipulation
- SIEM blind spot identification
- Covering tracks: bash history manipulation, log truncation
- Using legitimate admin tools to reduce noise

---

## 10. Active Directory Attack Paths

- Full BloodHound path analysis and shortest-path exploitation
- Domain privilege escalation via ACL abuse (GenericAll, WriteDACL, etc.)
- AdminSDHolder abuse
- Group Policy Object (GPO) abuse for code execution
- Cross-domain and cross-forest trust exploitation
- Azure AD / Entra ID hybrid attack paths
- Password policy enumeration and targeted spraying
- RODC (Read-Only Domain Controller) key list attack
- SID history injection

---

## 11. Cloud Red Teaming

### AWS
- IAM privilege escalation paths (PassRole, CreateRole, etc.)
- EC2 metadata service exploitation
- Lambda function abuse
- S3 bucket policy misconfiguration exploitation
- AWS credential exfiltration and pivoting via CLI/SDK

### Azure / Entra ID
- Service principal and app registration abuse
- Managed identity exploitation
- Azure DevOps pipeline poisoning
- Conditional Access policy bypass
- PRT (Primary Refresh Token) theft and replay

### GCP
- Service account key theft and impersonation
- Workload Identity Federation abuse
- GCS bucket public exposure exploitation

---

## 12. Reporting & Documentation

- Real-time note-taking and evidence collection (screenshots, logs, timestamps)
- Structured finding documentation (title, CVSS score, description, impact, PoC, remediation)
- Attack path narrative writing for executive audiences
- Technical remediation guidance for engineering teams
- MITRE ATT&CK framework mapping for all TTPs
- Risk rating methodology (CVSS v3.1, custom risk matrices)
- Deconfliction log maintenance for safe-to-operate windows
- Final report writing: executive summary, technical findings, appendices
- Lessons learned and purple team handoff preparation

---

## Tools Quick Reference

| Category | Tools |
|---|---|
| Recon | Amass, subfinder, theHarvester, Shodan, FOFA |
| Scanning | nmap, masscan, nuclei, feroxbuster, ffuf |
| AD Attack | BloodHound, CrackMapExec, Impacket, Rubeus, Mimikatz |
| Exploitation | Metasploit, sqlmap, Burp Suite Pro |
| C2 | Cobalt Strike, Sliver, Havoc, Brute Ratel C4, Mythic |
| Pivoting | Chisel, Ligolo-ng, proxychains, SSH, frp |
| Evasion | Donut, Scarecrow, PE2Shellcode, custom loaders |
| Cloud | Pacu, Prowler, ScoutSuite, ROADtools, AADInternals |
| Password | Hashcat, John the Ripper, Spray, MSOLSpray |

---

*Maintained for authorized red team engagements only. All activities must be conducted within the scope of a signed Rules of Engagement (ROE) document.*
