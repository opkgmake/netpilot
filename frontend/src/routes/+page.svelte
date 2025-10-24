<script>
  import { onMount } from 'svelte';

  // --- State Management ---
  let status = { isLoading: true, message: '初始化中...', isError: false, rule: null };
  let selectedInterface = '', inputBandwidth = 50, availableInterfaces = [];
  let selectedAlgorithm = 'cake';
  const algorithms = [
    { value: 'cake', label: 'CAKE（推荐）', needsBandwidth: true },
    { value: 'fq_codel', label: 'FQ Codel（公平队列）', needsBandwidth: false },
    { value: 'sfq', label: 'SFQ（随机公平队列）', needsBandwidth: false },
    { value: 'tbf', label: 'TBF（简单限速器）', needsBandwidth: true },
  ];
  const algorithmLabelMap = algorithms.reduce((acc, item) => {
    acc[item.value] = item.label;
    return acc;
  }, {});
  $: selectedAlgorithmLabel = algorithmLabelMap[selectedAlgorithm] || selectedAlgorithm;
  let showAdvanced = false;

  // --- API Functions (Defined only ONCE) ---

  async function querySystemState(iface) {
    if (!iface) return;
    status = { isLoading: true, message: `正在检查 ${iface} 的状态...`, isError: false, rule: null };
    try {
      const response = await fetch(`/api/qos/rules?interface=${iface}`);
      if (!response.ok && response.status !== 204) {
        const errorText = await response.text();
        throw new Error(`接口错误: ${response.status} - ${errorText}`);
      }
      
      if (response.status === 204) {
        status = { isLoading: false, message: `在 ${iface} 上没有由 NetPilot 管理的活动规则。`, isError: false, rule: null };
      } else {
        const rule = await response.json();
        status = { isLoading: false, message: `在 ${iface} 上检测到活动规则。`, isError: false, rule: rule };
        if (rule.algorithm && algorithms.some(a => a.value === rule.algorithm)) {
          selectedAlgorithm = rule.algorithm;
        }
        if (rule.settings && rule.settings.bandwidth) {
          inputBandwidth = parseInt(rule.settings.bandwidth) || 50;
        } else if (rule.settings && rule.settings.rate) {
          inputBandwidth = parseInt(rule.settings.rate) || 50;
        }
      }
    } catch (error) {
      console.error('获取规则失败:', error);
      status = { isLoading: false, message: error.message, isError: true, rule: null };
    }
  }

  async function applyRule() {
    if (!selectedInterface) return;
    const currentAlgo = algorithms.find(a => a.value === selectedAlgorithm);
    if (currentAlgo && currentAlgo.needsBandwidth && (!inputBandwidth || inputBandwidth <= 0)) {
       status = { ...status, isLoading: false, message: '需要填写正数带宽。', isError: true };
       return;
    }
    status = { isLoading: true, message: `正在应用「${selectedAlgorithmLabel}」规则...`, isError: false, rule: status.rule };
    try {
      const response = await fetch('/api/qos/rules', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          interface: selectedInterface,
          algorithm: selectedAlgorithm,
          settings: { bandwidth_mbit: Number(inputBandwidth) },
        }),
      });
      const result = await response.json();
      if (!response.ok) throw new Error(result.message || '应用规则失败');
      await querySystemState(selectedInterface);
    } catch (error) {
      console.error('应用规则出错:', error);
      status = { isLoading: false, message: error.message, isError: true, rule: null };
    }
  }

  async function resetToDefault() {
    if (!selectedInterface) return;
    status = { isLoading: true, message: `正在重置 ${selectedInterface} 的 QoS...`, isError: false, rule: status.rule };
    try {
      const response = await fetch(`/api/qos/rules?interface=${selectedInterface}`, { method: 'DELETE' });
      const result = await response.json();
      if (!response.ok) throw new Error(result.message || '删除规则失败');
      await querySystemState(selectedInterface);
    } catch (error) {
      console.error('删除规则出错:', error);
      status = { isLoading: false, message: error.message, isError: true, rule: null };
    }
  }

  // --- NEW: A helper function to parse CAKE's complex options string ---
  function parseCakeOptions(optionsStr) {
    if (!optionsStr) return null;
    const options = optionsStr.split(/\s+/);
    const details = {
      '隔离': [],
      'DSCP 处理': [],
      'ACK 过滤': [],
      '特殊功能': [],
      '时延与开销': [],
    };

    for (let i = 0; i < options.length; i++) {
      const opt = options[i];
      if (opt.includes('isolate')) details['隔离'].push(opt);
      else if (opt.includes('diffserv')) details['DSCP 处理'].push(opt);
      else if (opt.includes('wash')) details['DSCP 处理'].push(opt);
      else if (opt.includes('ack-filter')) details['ACK 过滤'].push(opt);
      else if (opt.includes('gso')) details['特殊功能'].push(opt);
      else if (opt === 'nat' || opt === 'nonat') details['隔离'].push(opt);
      else if (opt === 'rtt' || opt === 'overhead') {
        details['时延与开销'].push(`${opt} ${options[++i]}`);
      }
    }
    return details;
  }

  // --- NEW: A computed property that automatically parses cake options ---
  $: cakeDetails = status.rule?.algorithm === 'cake' ? parseCakeOptions(status.rule.settings.options) : null;
  
  // --- Lifecycle & Reactivity ---
  onMount(async () => {
    status = { isLoading: true, message: '正在获取网络接口...', isError: false, rule: null };
    try {
      const response = await fetch('/api/interfaces');
      if (!response.ok) throw new Error('获取网络接口失败');
      availableInterfaces = await response.json();
      if (availableInterfaces.length > 0) {
        selectedInterface = availableInterfaces[0];
      } else {
        status = { isLoading: false, message: '未发现任何网络接口。', isError: true, rule: null };
      }
    } catch (error) {
      console.error('获取接口出错:', error);
      status = { isLoading: false, message: error.message, isError: true, rule: null };
    }
  });
  
  $: if (selectedInterface) {
    querySystemState(selectedInterface);
  }

</script>

<main class="min-h-screen bg-gray-100 flex items-center justify-center p-4">
  <div class="w-full max-w-2xl bg-white rounded-xl shadow-lg p-8 space-y-6">
    
    <!-- ... (Header and Configuration sections are perfect, no changes needed) ... -->
    <header class="text-center">
      <h1 class="text-3xl font-bold text-gray-900 flex items-center justify-center gap-2">✈️ NetPilot</h1>
      <p class="text-gray-500 mt-1">Linux QoS 可视化控制台</p>
    </header>
    <div class="border-t"></div>
    <section class="space-y-4">
      <div>
        <label for="interface-select" class="block text-sm font-medium text-gray-700">网络接口</label>
        <select id="interface-select" bind:value={selectedInterface} class="mt-1 block w-full pl-3 pr-10 py-2 border-gray-300 rounded-md" disabled={availableInterfaces.length === 0}>
            {#if availableInterfaces.length === 0}<option value="">加载中...</option>{:else}{#each availableInterfaces as iface}<option value={iface}>{iface}</option>{/each}{/if}
        </select>
      </div>
      <div>
        <label for="algorithm-select" class="block text-sm font-medium text-gray-700">选择要应用的 QoS 算法</label>
        <select id="algorithm-select" bind:value={selectedAlgorithm} class="mt-1 block w-full pl-3 pr-10 py-2 border-gray-300 rounded-md">
          {#each algorithms as algo}<option value={algo.value}>{algo.label}</option>{/each}
        </select>
      </div>
      {#if algorithms.find(a => a.value === selectedAlgorithm)?.needsBandwidth}
        <div>
          <label for="bandwidth-input" class="block text-sm font-medium text-gray-700">带宽（Mbit/s）</label>
          <input type="number" id="bandwidth-input" bind:value={inputBandwidth} class="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md" min="1">
        </div>
      {/if}
      <div class="relative flex items-start pt-2">
        <div class="flex items-center h-5">
          <input id="advanced-toggle" type="checkbox" bind:checked={showAdvanced} class="focus:ring-indigo-500 h-4 w-4 text-indigo-600 border-gray-300 rounded">
        </div>
        <div class="ml-3 text-sm">
          <label for="advanced-toggle" class="font-medium text-gray-700">显示高级选项</label>
        </div>
      </div>
      {#if showAdvanced}
        <div class="mt-4 p-4 border-l-4 border-yellow-400 bg-yellow-50"><p class="text-sm text-yellow-700">高级选项尚未实现。</p></div>
      {/if}
    </section>

    <!-- 【核心进化】Status Section - now with beautiful, structured CAKE details -->
    <section class="bg-gray-50 p-4 rounded-lg min-h-[150px] flex flex-col justify-center">
      <h3 class="text-center font-semibold text-gray-700 mb-3">当前接口 {selectedInterface || '...'} 的状态</h3>
      <div class="text-left text-sm">
        {#if status.isLoading}
          <p class="text-center text-yellow-600">{status.message}</p>
        {:else if status.isError}
          <p class="text-center text-red-600 font-semibold">{status.message}</p>
        {:else if status.rule && status.rule.algorithm}
          <div class="space-y-2">
            <div class="flex items-center">
              <span class="w-28 font-semibold text-gray-600">算法:</span>
              <span class="px-2 py-1 bg-green-100 text-green-800 text-xs font-medium rounded-full">{algorithmLabelMap[status.rule.algorithm] || status.rule.algorithm}</span>
            </div>

            <!-- This is the "smart" part -->
            {#if status.rule.algorithm === 'cake' && cakeDetails}
              <div class="pt-2">
                <h4 class="font-semibold text-gray-600 mb-1">参数:</h4>
                <div class="pl-4 border-l-2 border-gray-200 space-y-1">
                  <div class="flex"><span class="w-24 text-gray-500">带宽:</span><code class="text-indigo-700">{status.rule.settings.bandwidth}</code></div>
                  {#each Object.entries(cakeDetails) as [category, opts]}
                    {#if opts.length > 0}
                      <div class="flex"><span class="w-24 text-gray-500">{category}:</span><code class="text-indigo-700">{opts.join(' ')}</code></div>
                    {/if}
                  {/each}
                </div>
              </div>
            {:else if status.rule.settings && Object.keys(status.rule.settings).length > 0}
              <!-- Fallback for other algorithms like TBF, SFQ -->
              <div class="pt-2">
                <h4 class="font-semibold text-gray-600 mb-1">参数:</h4>
                <div class="pl-4 border-l-2 border-gray-200 space-y-1">
                  {#each Object.entries(status.rule.settings) as [key, value]}
                    <div class="flex items-baseline">
                      <span class="w-24 text-gray-500 capitalize">{key.replace('_', ' ')}:</span>
                      <code class="text-indigo-700">{value}</code>
                    </div>
                  {/each}
                </div>
              </div>
            {/if}
          </div>
        {:else}
          <p class="text-center text-blue-700">{status.message || `系统默认设置生效。`}</p>
        {/if}
      </div>
    </section>

    <!-- ... (Buttons section is perfect, no changes needed) ... -->
    <section class="pt-2 flex flex-col sm:flex-row gap-3">
      <button on:click={applyRule} class="w-full py-2 px-4 rounded-md font-medium text-white bg-indigo-600 hover:bg-indigo-700" disabled={status.isLoading}>应用「{selectedAlgorithmLabel}」</button>
      <button on:click={resetToDefault} class="w-full py-2 px-4 rounded-md font-medium text-white bg-gray-500 hover:bg-gray-600" disabled={status.isLoading || !status.rule}>恢复默认</button>
    </section>
  </div>
</main>
