<?php

namespace App\Http\Controllers;

use App\Models\Olt;
use App\Models\Onu;
use App\Services\ZteTelnetService;
use Illuminate\Http\Request;

class OnuController extends Controller
{
    public function index(Request $request)
    {
        $query = Onu::with('olt');

        if ($request->filled('search')) {
            $search = $request->search;
            $query->where(function ($q) use ($search) {
                $q->where('name', 'like', "%{$search}%")
                    ->orWhere('sn', 'like', "%{$search}%")
                    ->orWhere('type', 'like', "%{$search}%")
                    ->orWhereHas('olt', function ($q) use ($search) {
                        $q->where('name', 'like', "%{$search}%")
                            ->orWhere('ip', 'like', "%{$search}%");
                    });
            });
        }

        $onus = $query->latest()->paginate(20)->withQueryString();
        $totalDbOnusCount = \App\Models\Onu::count();

        return view('onus.index', compact('onus', 'totalDbOnusCount'));
    }

    public function syncBackground()
    {
        try {
            $olts = Olt::all();
            $countBefore = Onu::count();

            foreach ($olts as $olt) {
                try {
                    // === FAST: SNMP BulkWalk ===
                    $synced = $this->syncViaSNMP($olt);
                    if ($synced > 0) continue; // SNMP berhasil, skip Telnet
                } catch (\Throwable $e) {
                    // SNMP gagal, fallback ke Telnet
                }

                // === FALLBACK: Telnet ===
                try {
                    $telnet = new ZteTelnetService();
                    $telnet->syncOnus($olt);
                } catch (\Throwable $e) {
                    // Skip
                }
            }

            $countAfter = Onu::count();
            $newOnus = $countAfter - $countBefore;

            return response()->json(['success' => true, 'count' => $newOnus]);
        } catch (\Exception $e) {
            return response()->json(['success' => false, 'error' => $e->getMessage()]);
        }
    }

    /**
     * Sync ONUs via SNMP BulkWalk — secepat kilat!
     * 
     * OID Discovery Results (ZTE C320 V2.1.0):
     * - SN (Hex-STRING 8 bytes):  .1.3.6.1.4.1.3902.1012.3.50.11.2.1.3.{ponIfIndex}.{onuId}
     * - Name (STRING):            .1.3.6.1.4.1.3902.1012.3.28.1.1.2.{ponIfIndex}.{onuId}
     * - Vendor ID (STRING 4chr):  .1.3.6.1.4.1.3902.1012.3.50.11.2.1.1.{ponIfIndex}.{onuId}
     * 
     * ponIfIndex = 268500992 + port*256 (board 1, slot 2)
     */
    private function syncViaSNMP(Olt $olt): int
    {
        if (!extension_loaded('snmp')) return 0;

        $snmpVersion = $olt->snmp_version ?? 'v2c';
        $target = "{$olt->ip}:{$olt->snmp_port}";

        // === OID Trees (verified on ZTE C320 V2.1.0) ===
        $snOidBase   = ".1.3.6.1.4.1.3902.1012.3.50.11.2.1.3";  // SN (Hex-STRING)
        $nameOidBase = ".1.3.6.1.4.1.3902.1012.3.28.1.1.2";     // Name/Description
        $typeOidBase = ".1.3.6.1.4.1.3902.1012.3.50.11.2.1.1";  // Vendor ID (ZTEG/HWTC)

        // === Setup SNMP ===
        if ($snmpVersion === 'v3') {
            $snmpUser = !empty($olt->snmp_username) ? $olt->snmp_username : 'oltuser';
            $snmpPass = !empty($olt->snmp_password) ? $olt->snmp_password : '';
            $snmp = new \SNMP(\SNMP::VERSION_3, $target, $snmpUser, 15000000, 2);
            $snmp->setSecurity('authNoPriv', 'MD5', $snmpPass);
        } else {
            $community = !empty($olt->snmp_username) ? $olt->snmp_username : 'public';
            $snmp = new \SNMP(\SNMP::VERSION_2c, $target, $community, 15000000, 2);
        }

        $snmp->valueretrieval = SNMP_VALUE_PLAIN;
        $snmp->oid_output_format = SNMP_OID_OUTPUT_NUMERIC;
        $snmp->exceptions_enabled = 0;
        $snmp->max_oids = 50; // GETBULK: 50 OID per request

        // === 1. BulkWalk Serial Numbers ===
        $snResults = @$snmp->walk($snOidBase);
        if ($snResults === false || empty($snResults)) {
            @$snmp->close();
            return 0;
        }

        // === 2. BulkWalk Names ===
        $nameResults = @$snmp->walk($nameOidBase) ?: [];

        // === 3. BulkWalk Vendor/Type ===
        $typeResults = @$snmp->walk($typeOidBase) ?: [];

        @$snmp->close();

        // === Build lookup maps (key = "ponIfIndex.onuId") ===
        $nameMap = $this->buildIndexMap($nameResults, 2);
        $typeMap = $this->buildIndexMap($typeResults, 2);

        // === Parse SN results & save ===
        $syncedCount = 0;

        foreach ($snResults as $oid => $snRaw) {
            // OID: ...3.{ponIfIndex}.{onuId}
            $parts = explode('.', $oid);
            $len = count($parts);
            if ($len < 2) continue;

            $ponIfIndex = (int)$parts[$len - 2];
            $onuId = (int)$parts[$len - 1];
            if ($onuId <= 0) continue;

            // === Decode board/slot/port dari ponIfIndex ===
            // Formula: ponIfIndex = 0x10000000 + slot*0x10000 + port*0x100
            // Contoh: 268501248 = 0x10010100 → slot=1, port=1
            //         268566784 = 0x10020100 → slot=2, port=1
            $board = 1; // ZTE C320 selalu board 1
            $base = 268435456; // 0x10000000
            $offset = $ponIfIndex - $base;
            if ($offset < 0) continue;

            $slot = intdiv($offset, 65536);   // 0x10000
            $port = intdiv($offset % 65536, 256); // 0x100

            if ($slot <= 0 || $slot > 20 || $port <= 0 || $port > 128) continue;

            // === Parse SN (8 bytes: 4 ASCII vendor + 4 hex suffix) ===
            $sn = $this->parseHexSN($snRaw);
            if (empty($sn) || strlen($sn) < 8) continue;

            // === Get name & type ===
            $indexKey = "{$ponIfIndex}.{$onuId}";
            $name = $nameMap[$indexKey] ?? $sn;
            $vendorType = $typeMap[$indexKey] ?? 'Unknown';

            Onu::updateOrCreate(
                ['sn' => $sn],
                [
                    'olt_id' => $olt->id,
                    'board' => $board,
                    'slot' => $slot,
                    'port' => $port,
                    'onu_index' => $onuId,
                    'type' => $vendorType,
                    'name' => !empty($name) ? $name : $sn,
                ]
            );
            $syncedCount++;
        }

        return $syncedCount;
    }

    /**
     * Build index map dari SNMP walk results
     * @param int $depth berapa level dari belakang untuk key (2 = "x.y", 3 = "x.y.z")
     */
    private function buildIndexMap(array $results, int $depth = 2): array
    {
        $map = [];
        foreach ($results as $oid => $value) {
            $parts = explode('.', $oid);
            $len = count($parts);
            if ($len < $depth) continue;
            $keyParts = array_slice($parts, -$depth);
            $key = implode('.', $keyParts);
            $map[$key] = trim($value, " \t\n\r\0\x0B\"");
        }
        return $map;
    }

    /**
     * Parse Hex-STRING SN (8 bytes) ke format readable
     * Input: raw 8 bytes (4 ASCII vendor + 4 binary suffix)
     * Output: "HWTC09F8019A" format
     */
    private function parseHexSN(string $raw): string
    {
        // SNMP_VALUE_PLAIN returns raw binary for Hex-STRING
        if (strlen($raw) === 8) {
            $vendor = substr($raw, 0, 4);
            $suffix = strtoupper(bin2hex(substr($raw, 4, 4)));
            // Verify vendor is ASCII printable
            if (ctype_alpha($vendor)) {
                return $vendor . $suffix;
            }
        }

        // Fallback: might be already formatted string
        $raw = trim($raw, " \t\n\r\0\x0B\"");

        // Handle "XX XX XX XX XX XX XX XX" hex format
        if (preg_match('/^([0-9a-fA-F]{2}[\s:]){7}[0-9a-fA-F]{2}$/', $raw)) {
            $bytes = pack('H*', preg_replace('/[\s:]/', '', $raw));
            $vendor = substr($bytes, 0, 4);
            $suffix = strtoupper(bin2hex(substr($bytes, 4, 4)));
            if (ctype_alpha($vendor)) {
                return $vendor . $suffix;
            }
        }

        // Already a plain SN string
        if (preg_match('/^[A-Z]{4}[0-9A-F]{8}$/i', $raw)) {
            return strtoupper($raw);
        }

        return $raw;
    }

    public function unconfigured()
    {
        $olts = Olt::all();
        $unconfigured = [];
        $error = null;

        $oltId = request('olt_id', $olts->first()->id ?? null);
        $selectedOlt = null;

        if ($oltId) {
            $selectedOlt = Olt::find($oltId);
            if ($selectedOlt) {
                try {
                    $telnet = new ZteTelnetService();
                    $telnet->connect(
                        $selectedOlt->ip,
                        $selectedOlt->telnet_port,
                        $selectedOlt->telnet_username,
                        $selectedOlt->telnet_password
                    );
                    $unconfigured = $telnet->getUnconfiguredOnus();
                    $telnet->disconnect();
                } catch (\Exception $e) {
                    $error = "Failed to connect to OLT: " . $e->getMessage();
                }
            }
        }

        return view('onus.unconfigured', compact('olts', 'unconfigured', 'selectedOlt', 'error'));
    }

    public function create(Request $request)
    {
        $olts = Olt::all();
        $prefill = [
            'olt_id' => $request->get('olt_id'),
            'board' => $request->get('board', '1'),
            'slot' => $request->get('slot', '1'),
            'port' => $request->get('port', '1'),
            'onu_index' => $request->get('onu_index'),
            'sn' => $request->get('sn'),
        ];

        $tcontProfiles = [];
        $trafficProfiles = [];

        if (!empty($prefill['olt_id'])) {
            $olt = Olt::find($prefill['olt_id']);
            if ($olt) {
                try {
                    $telnet = new ZteTelnetService();
                    $telnet->connect($olt->ip, $olt->telnet_port, $olt->telnet_username, $olt->telnet_password);

                    // Get TCONT (Upstream) Profiles
                    $tcontOutput = $telnet->execute("show gpon profile tcont");
                    if (preg_match_all('/Profile name\s*:\s*(\S+)/i', $tcontOutput, $matches)) {
                        $tcontProfiles = $matches[1];
                    }

                    // Get Traffic (Downstream) Profiles
                    $trafficOutput = $telnet->execute("show gpon profile traffic");
                    if (preg_match_all('/Profile name\s*:\s*(\S+)/i', $trafficOutput, $matches)) {
                        $trafficProfiles = $matches[1];
                    }

                    // Get Available ONU Indices
                    $availableIndices = [];
                    $usedIndices = [];
                    $oltInterfaceConfig = $telnet->execute("show running-config interface gpon-olt_{$prefill['board']}/{$prefill['slot']}/{$prefill['port']}");
                    if (preg_match_all('/onu\s+(\d+)\s+type/i', $oltInterfaceConfig, $matches)) {
                        $usedIndices = array_map('intval', $matches[1]);
                    }
                    for ($i = 1; $i <= 128; $i++) {
                        if (!in_array($i, $usedIndices)) {
                            $availableIndices[] = $i;
                        }
                    }

                    $telnet->disconnect();
                } catch (\Exception $e) {
                    // Fail silently
                }
            }
        }

        $acsProfiles = \App\Models\AcsProfile::all();
        $defaultAcs = \App\Models\AcsProfile::where('is_default', true)->first();

        $scriptTemplates = \App\Models\ScriptTemplate::all();

        $sn = $prefill['sn'] ?? '';
        $defaultTemplate = null;

        if (str_starts_with($sn, 'FHTT')) {
            $defaultTemplate = \App\Models\ScriptTemplate::where('merk', 'like', '%Fiberhome%')->orWhere('merk', 'like', '%FHTT%')->first();
        } elseif (str_starts_with($sn, 'HWTC')) {
            $defaultTemplate = \App\Models\ScriptTemplate::where('merk', 'like', '%Huawei%')->orWhere('merk', 'like', '%HWTC%')->first();
        } elseif (str_starts_with($sn, 'ZTEG')) {
            $defaultTemplate = \App\Models\ScriptTemplate::where('merk', 'like', '%ZTE%')->first();
        }

        if (!$defaultTemplate) {
            $defaultTemplate = \App\Models\ScriptTemplate::where('is_default', true)->first();
        }

        return view('onus.create', compact('olts', 'prefill', 'tcontProfiles', 'trafficProfiles', 'availableIndices', 'acsProfiles', 'defaultAcs', 'scriptTemplates', 'defaultTemplate'));
    }

    public function store(Request $request)
    {
        $validated = $request->validate([
            'olt_id' => 'required|exists:olts,id',
            'board' => 'required|string',
            'slot' => 'required|string',
            'port' => 'required|string',
            'onu_index' => 'required|string',
            'sn' => 'required|string',
            'name' => 'required|string',
            'type' => 'required|string',
            'upstream_profile' => 'nullable|string',
            'downstream_profile' => 'nullable|string',
            'raw_gpon_onu' => 'required|string',
            'raw_pon_onu_mng' => 'required|string',
        ]);

        $olt = Olt::findOrFail($validated['olt_id']);

        try {
            $telnet = new ZteTelnetService();
            $telnet->connect(
                $olt->ip,
                $olt->telnet_port,
                $olt->telnet_username,
                $olt->telnet_password
            );

            $telnet->provisionOnu(
                $validated['board'],
                $validated['slot'],
                $validated['port'],
                $validated['onu_index'],
                $validated['type'],
                $validated['sn'],
                $validated['name'],
                $validated['raw_gpon_onu'],
                $validated['raw_pon_onu_mng']
            );

            $telnet->disconnect();

            Onu::create($validated);

            return redirect()->route('onus.index')->with('success', 'ONU provisioned successfully');
        } catch (\Exception $e) {
            return back()->with('error', 'Provisioning failed: ' . $e->getMessage())->withInput();
        }
    }

    public function edit(Onu $onu)
    {
        $olt = $onu->olt;
        $configOnu = "Konfigurasi tidak ditemukan atau gagal mengambil dari OLT.";
        $configMng = "Konfigurasi tidak ditemukan atau gagal mengambil dari OLT.";

        $tconts = [];
        $gemports = [];
        $servicePorts = [];

        try {
            $telnet = new ZteTelnetService();
            $telnet->connect($olt->ip, $olt->telnet_port, $olt->telnet_username, $olt->telnet_password);

            // Get full config to parse
            $fullConfig = $telnet->execute("show running-config");
            $telnet->disconnect();

            // Parse interface gpon-onu
            $onuInterface = "interface gpon-onu_{$onu->board}/{$onu->slot}/{$onu->port}:{$onu->onu_index}";
            if (preg_match("~^$onuInterface\r?\n(.*?)\r?\n!~sm", $fullConfig, $matches)) {
                $rawOnu = trim($matches[1]);
                $configOnu = $onuInterface . "\n" . $rawOnu . "\n!";

                // Extract TCONTs
                if (preg_match_all("/tcont\s+(\d+)\s+name\s+(\S+)\s+profile\s+(\S+)/i", $rawOnu, $tcontMatches, PREG_SET_ORDER)) {
                    foreach ($tcontMatches as $m) {
                        $tconts[$m[1]] = ['name' => $m[2], 'profile' => $m[3]];
                    }
                }

                // Extract Gemports
                if (preg_match_all("/gemport\s+(\d+)\s+tcont\s+(\d+)/i", $rawOnu, $gemMatches, PREG_SET_ORDER)) {
                    foreach ($gemMatches as $m) {
                        $gemports[$m[1]] = ['tcont' => $m[2], 'traffic_limit' => ''];
                    }
                }
                if (preg_match_all("/gemport\s+(\d+)\s+traffic-limit\s+downstream\s+(\S+)/i", $rawOnu, $tlMatches, PREG_SET_ORDER)) {
                    foreach ($tlMatches as $m) {
                        if (isset($gemports[$m[1]])) {
                            $gemports[$m[1]]['traffic_limit'] = $m[2];
                        }
                    }
                }

                // Extract Service Ports
                if (preg_match_all("/service-port\s+(\d+)\s+vport\s+(\d+)\s+user-vlan\s+(\d+)\s+vlan\s+(\d+)/i", $rawOnu, $spMatches, PREG_SET_ORDER)) {
                    foreach ($spMatches as $m) {
                        $servicePorts[$m[1]] = ['vport' => $m[2], 'user_vlan' => $m[3], 'vlan' => $m[4]];
                    }
                }
            }

            // Parse pon-onu-mng
            $mngInterface = "pon-onu-mng gpon-onu_{$onu->board}/{$onu->slot}/{$onu->port}:{$onu->onu_index}";
            if (preg_match("~^$mngInterface\r?\n(.*?)\r?\n!~sm", $fullConfig, $matches)) {
                $configMng = $mngInterface . "\n" . trim($matches[1]) . "\n!";
            }
        } catch (\Exception $e) {
            $configOnu = "Error: " . $e->getMessage();
        }

        return view('onus.edit', compact('onu', 'configOnu', 'configMng', 'tconts', 'gemports', 'servicePorts'));
    }

    public function update(Request $request, Onu $onu)
    {
        $validated = $request->validate([
            'name' => 'required|string|max:255',
            'tconts' => 'nullable|array',
            'gemports' => 'nullable|array',
            'service_ports' => 'nullable|array',
        ]);

        $olt = $onu->olt;

        try {
            $telnet = new ZteTelnetService();
            $telnet->connect(
                $olt->ip,
                $olt->telnet_port,
                $olt->telnet_username,
                $olt->telnet_password
            );

            $telnet->execute("conf t");
            $telnet->execute("interface gpon-onu_{$onu->board}/{$onu->slot}/{$onu->port}:{$onu->onu_index}");

            // Update Name
            $telnet->execute("name {$validated['name']}");

            // Update TCONTs
            if (!empty($validated['tconts'])) {
                foreach ($validated['tconts'] as $id => $data) {
                    if (!empty($data['profile'])) {
                        $telnet->execute("tcont {$id} profile {$data['profile']}");
                    }
                }
            }

            // Update Gemports Traffic Limits
            if (!empty($validated['gemports'])) {
                foreach ($validated['gemports'] as $id => $data) {
                    if (!empty($data['traffic_limit'])) {
                        $telnet->execute("gemport {$id} traffic-limit downstream {$data['traffic_limit']}");
                    }
                }
            }

            // Update Service Ports / VLANs
            if (!empty($validated['service_ports'])) {
                foreach ($validated['service_ports'] as $id => $data) {
                    if (!empty($data['user_vlan']) && !empty($data['vlan']) && !empty($data['vport'])) {
                        // First remove the old service port, then recreate it to prevent errors in ZTE C320
                        $telnet->execute("no service-port {$id}");
                        $telnet->execute("service-port {$id} vport {$data['vport']} user-vlan {$data['user_vlan']} vlan {$data['vlan']}");
                    }
                }
            }

            // Advanced Raw CLI Execution for gpon-onu
            if ($request->filled('raw_gpon_onu')) {
                $lines = explode("\n", $request->input('raw_gpon_onu'));
                foreach ($lines as $line) {
                    $cmd = trim($line);
                    if ($cmd !== '' && strpos($cmd, 'name ') !== 0) { // Skip name as it's handled above
                        $telnet->execute($cmd);
                    }
                }
            }

            $telnet->execute("exit"); // Exit gpon-onu

            // Advanced Raw CLI Execution for pon-onu-mng
            if ($request->filled('raw_pon_onu_mng')) {
                $telnet->execute("pon-onu-mng gpon-onu_{$onu->board}/{$onu->slot}/{$onu->port}:{$onu->onu_index}");
                $lines = explode("\n", $request->input('raw_pon_onu_mng'));
                foreach ($lines as $line) {
                    $cmd = trim($line);
                    if ($cmd !== '') {
                        $telnet->execute($cmd);
                    }
                }
                $telnet->execute("exit"); // Exit pon-onu-mng
            }

            $telnet->execute("exit"); // Exit config t

            $telnet->disconnect();

            $onu->update(['name' => $validated['name']]);

            return redirect()->route('onus.index')->with('success', 'ONU configuration updated successfully');
        } catch (\Exception $e) {
            return back()->with('error', 'Update failed: ' . $e->getMessage())->withInput();
        }
    }

    public function destroy(Onu $onu)
    {
        $olt = $onu->olt;

        try {
            $telnet = new ZteTelnetService();
            $telnet->connect(
                $olt->ip,
                $olt->telnet_port,
                $olt->telnet_username,
                $olt->telnet_password
            );

            $telnet->unprovisionOnu(
                $onu->board,
                $onu->slot,
                $onu->port,
                $onu->onu_index
            );

            $telnet->disconnect();

            $onu->delete();

            return redirect()->route('onus.index')->with('success', 'ONU unprovisioned successfully');
        } catch (\Exception $e) {
            return back()->with('error', 'Unprovisioning failed: ' . $e->getMessage());
        }
    }

    public function power(Onu $onu)
    {
        // === Cache 30 detik per ONU — tidak query SNMP berulang ===
        $cacheKey = "onu_power_{$onu->id}";
        $cacheTtl = 30; // detik

        $cached = \Illuminate\Support\Facades\Cache::get($cacheKey);
        if ($cached) {
            return response()->json($cached);
        }

        try {
            $olt = $onu->olt;
            if (!$olt) {
                throw new \Exception("OLT tidak ditemukan");
            }

            $rxOlt = 'N/A';
            $rxOnu = 'N/A';
            $snmpVersion = $olt->snmp_version ?? 'v2c';
            $method = 'SNMP ' . strtoupper($snmpVersion);

            if (!extension_loaded('snmp')) {
                throw new \Exception("Ekstensi SNMP PHP belum diaktifkan.");
            }

            // === Hitung OID ===
            $slot = (int)$onu->slot;
            $pon = (int)$onu->port;
            $onuId = (int)$onu->onu_index;

            // RX Power tree: base per GPON card slot + port
            $baseOnuID = 285278208 + ($slot * 256);
            $onuIDSuffix = $baseOnuID + $pon;
            $rxOnuOid = ".1.3.6.1.4.1.3902.1082.500.20.2.2.2.1.10.{$onuIDSuffix}.{$onuId}.1";

            // RX OLT OID (upstream power)
            $rxOltOid = ".1.3.6.1.4.1.3902.1082.500.20.2.2.2.1.10.{$onuIDSuffix}.{$onuId}.2";

            // === SNMP Query ===
            $target = "{$olt->ip}:{$olt->snmp_port}";
            $timeout = 500000;
            $retries = 1;

            if ($snmpVersion === 'v3') {
                $snmpUser = !empty($olt->snmp_username) ? $olt->snmp_username : 'oltuser';
                $snmpPass = !empty($olt->snmp_password) ? $olt->snmp_password : '';
                $snmp = new \SNMP(\SNMP::VERSION_3, $target, $snmpUser, $timeout, $retries);
                $snmp->setSecurity('authNoPriv', 'MD5', $snmpPass);
                $snmp->valueretrieval = SNMP_VALUE_PLAIN;
                $snmp->oid_output_format = SNMP_OID_OUTPUT_NUMERIC;
                $snmp->exceptions_enabled = 0;

                $rxOnuSnmp = @$snmp->get($rxOnuOid);
                $rxOltSnmp = @$snmp->get($rxOltOid);
                @$snmp->close();
            } else {
                $community = !empty($olt->snmp_username) ? $olt->snmp_username : 'public';
                snmp_set_valueretrieval(SNMP_VALUE_PLAIN);
                snmp_set_oid_output_format(SNMP_OID_OUTPUT_NUMERIC);
                $rxOnuSnmp = @snmpget($target, $community, $rxOnuOid, $timeout, $retries);
                $rxOltSnmp = @snmpget($target, $community, $rxOltOid, $timeout, $retries);
            }

            // Parse RX ONU
            if ($rxOnuSnmp !== false && $rxOnuSnmp !== null && (int)$rxOnuSnmp !== 0) {
                $rxOnuVal = ((int)$rxOnuSnmp * 0.002) - 30;
                $rxOnu = number_format($rxOnuVal, 2) . ' dBm';
            }

            // Parse RX OLT
            if ($rxOltSnmp !== false && $rxOltSnmp !== null && (int)$rxOltSnmp !== 0) {
                $rxOltVal = ((int)$rxOltSnmp * 0.002) - 30;
                $rxOlt = number_format($rxOltVal, 2) . ' dBm';
            }

            $result = [
                'success' => true,
                'rx_olt' => $rxOlt,
                'rx_onu' => $rxOnu,
                'method' => $method . ' (cached 30s)'
            ];

            // Simpan ke cache 30 detik
            \Illuminate\Support\Facades\Cache::put($cacheKey, $result, $cacheTtl);

            return response()->json($result);
        } catch (\Exception $e) {
            return response()->json([
                'success' => false,
                'message' => $e->getMessage()
            ], 500);
        }
    }
}
