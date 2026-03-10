// frontend/src/pages/LabDashboard.tsx
import { useState, useEffect } from 'react';
// Импорт типов, сгенерированных Wails (убедись, что пути верные)
import { 
  Material, 
  Standard, 
  TestMethod, 
  ExperimentGroup, 
  GroupSummary, 
  ProtocolListResponse,
  PaginatedMetadata
} from '../../wailsjs/go/models';

// Импорт функций API (имена должны совпадать с теми, что в internal/app/app.go)
import {
  GetMaterials,
  GetStandardsByMaterial,
  GetMethodsByStandard,
  CreateGroup,
  GetGroups,
  CreateProtocol,
  GetProtocols,
  GetGroupSummary,
  GenerateProtocolPDF,
  GenerateGroupSummaryPDF
} from '../../wailsjs/go/main/App';

import { SaveDialog } from '../../wailsjs/runtime/runtime';

// ============================================================================
// LOCAL TYPES & ADAPTERS
// ============================================================================

// Адаптер метода для удобства работы в UI (объединяем логику старой и новой схемы)
interface MethodUI extends TestMethod {
  is_calculated: boolean;
  formula: string;
  parameters: any[]; // inputs
  min_value?: number;
  max_value?: number;
}

interface ProtocolListItemUI {
  id: string;
  sample_number: string;
  lab_name: string;
  created_at: string;
  material_name: string;
  material_id: string;
}

export default function LabDashboard() {
  // === STATE: Справочники ===
  const [materials, setMaterials] = useState<Material[]>([]);
  const [standards, setStandards] = useState<Standard[]>([]); // Бывшие GOSTs
  const [methods, setMethods] = useState<MethodUI[]>([]);
  
  // === STATE: Формы ===
  const [labName, setLabName] = useState("БГТУ им В.Г. Шухова Кафедра материаловедения");
  const [operator, setOperator] = useState("Рулев Д.А.");
  const [project, setProject] = useState("-");
  
  const [sampleNumber, setSampleNumber] = useState("");
  const [samplePlace, setSamplePlace] = useState("");
  const [sampleNote, setSampleNote] = useState("");
  
  const [selectedGroupId, setSelectedGroupId] = useState<string>("");
  const [selectedMaterialId, setSelectedMaterialId] = useState<string>("");
  const [selectedStandardId, setSelectedStandardId] = useState<string>(""); // Бывший GOST
  const [selectedMethodId, setSelectedMethodId] = useState<string>("");
  
  // Ввод результатов
  const [simpleValue, setSimpleValue] = useState<string>("");
  const [rawInputs, setRawInputs] = useState<Record<string, string>>({});
  
  // === STATE: Списки и UI ===
  const [groups, setGroups] = useState<ExperimentGroup[]>([]);
  const [groupsMeta, setGroupsMeta] = useState<PaginatedMetadata | null>(null);
  const [groupsPage, setGroupsPage] = useState(1);
  
  const [protocols, setProtocols] = useState<ProtocolListItemUI[]>([]);
  const [protocolsMeta, setProtocolsMeta] = useState<PaginatedMetadata | null>(null);
  const [protocolsPage, setProtocolsPage] = useState(1);
  
  // Модалки
  const [isGroupModalOpen, setIsGroupModalOpen] = useState(false);
  const [newGroupName, setNewGroupName] = useState("");
  const [newGroupDesc, setNewGroupDesc] = useState(""); // В новой схеме может не использоваться напрямую
  const [newGroupMatId, setNewGroupMatId] = useState("");
  
  const [viewingProtocol, setViewingProtocol] = useState<ProtocolListItemUI | null>(null);
  const [viewingSummary, setViewingSummary] = useState<GroupSummary | null>(null);
  
  const [toast, setToast] = useState<{msg: string, error: boolean} | null>(null);
  const [loading, setLoading] = useState(false);

  const pageSize = 10;

  // === INITIALIZATION ===
  useEffect(() => {
    loadMaterials();
    loadGroupsList(1);
    loadProtocolsList(1);
  }, []);

  // === LOADERS ===
  const loadMaterials = async () => {
    try {
      const list = await GetMaterials();
      setMaterials(Array.isArray(list) ? list : []);
    } catch (e: any) { 
      showToast("Ошибка загрузки материалов: " + e.toString(), true); 
    }
  };

  const loadGroupsList = async (page: number) => {
    try {
      const offset = (page - 1) * pageSize;
      // Возвращает объект { items: [], meta: {} }
      const response: any = await GetGroups(pageSize, offset);
      
      setGroups(response.items || []);
      setGroupsMeta(response.meta || null);
      setGroupsPage(page);
    } catch (e: any) {
      console.error(e);
      showToast("Ошибка загрузки групп", true);
    }
  };

  const loadProtocolsList = async (page: number) => {
    try {
      const offset = (page - 1) * pageSize;
      const response: ProtocolListResponse = await GetProtocols(pageSize, offset);
      
      setProtocols((response.items as any) || []);
      setProtocolsMeta(response.meta || null);
      setProtocolsPage(page);
    } catch (e: any) {
      console.error(e);
      showToast("Ошибка загрузки протоколов", true);
    }
  };

  const loadStandards = async (matId: string) => {
    if (!matId) { setStandards([]); return; }
    try {
      const list = await GetStandardsByMaterial(matId);
      setStandards(Array.isArray(list) ? list : []);
      setMethods([]);
      setSelectedStandardId("");
      setSelectedMethodId("");
    } catch (e: any) { 
      showToast("Ошибка загрузки стандартов (ГОСТов)", true); 
    }
  };

  const loadMethods = async (standardId: string) => {
    if (!standardId) { setMethods([]); return; }
    try {
      const list = await GetMethodsByStandard(standardId);
      
      // Адаптируем новые модели под старый UI
      const adaptedMethods: MethodUI[] = (list || []).map(m => ({
        ...m,
        is_calculated: m.formula_expr !== "" && m.formula_expr !== null,
        formula: m.formula_expr || "",
        parameters: m.inputs || [], // inputs вместо parameters
        min_value: undefined, // В новой схеме нормы динамические, их нет в методе
        max_value: undefined
      }));

      setMethods(adaptedMethods);
      setSelectedMethodId("");
      setSimpleValue("");
      setRawInputs({});
    } catch (e: any) { 
      showToast("Ошибка загрузки методов", true); 
    }
  };

  // === HANDLERS ===
  const handleCreateGroup = async () => {
    if (!newGroupName || !newGroupMatId) {
      showToast("Заполните название и материал группы", true);
      return;
    }
    try {
      // В новой схеме: CreateGroup(name, project, location, materialID)
      await CreateGroup(newGroupName, newGroupDesc, "", newGroupMatId);
      setIsGroupModalOpen(false);
      setNewGroupName("");
      setNewGroupDesc("");
      setNewGroupMatId("");
      showToast("Группа создана");
      loadGroupsList(groupsPage);
    } catch (e: any) { 
      showToast("Ошибка: " + e.toString(), true); 
    }
  };

  const handleSaveProtocol = async () => {
    if (!sampleNumber || !samplePlace) { 
      showToast("Заполните номер пробы и место отбора", true); 
      return; 
    }
    // Материал теперь берется из группы или выбран явно, но в новой схеме создание без группы возможно
    // Если группы нет, материал должен быть выбран явно (добавь селект если нужно)
    if (!selectedMaterialId && !selectedGroupId) {
       showToast("Выберите материал или группу испытаний", true);
       return;
    }

    if (!selectedMethodId) { 
      showToast("Выберите метод", true); 
      return; 
    }
    
    const method = methods.find(m => m.id === selectedMethodId);
    if (!method) {
      showToast("Метод не найден", true);
      return;
    }

    // Формируем результаты в формате нового DTO
    // Структура зависит от того, как ты описал CreateProtocolRequest в models.go
    // Предположим, что там есть raw_inputs
    let resultsInput: any[] = [];

    if (method.is_calculated) {
      const inputParams = method.parameters.filter((p: any) => p.is_input);
      for (const p of inputParams) {
        const valStr = rawInputs[p.param_key]; // param_key вместо key
        if (p.is_required && (!valStr || valStr.trim() === '')) {
          showToast(`Заполните параметр: ${p.label}`, true);
          return;
        }
      }
      resultsInput.push({
        method_id: method.id,
        raw_inputs: rawInputs,
        note: undefined
      });
    } else {
      const val = parseFloat(simpleValue);
      if (isNaN(val)) { 
        showToast("Введите корректное число", true); 
        return; 
      }
      // Для простых методов передаем значение через raw_inputs с ключом "value"
      // Или адаптируй под свой DTO, если там есть поле value
      resultsInput.push({
        method_id: method.id,
        raw_inputs: { "value": val.toString() },
        note: undefined
      });
    }

    // Определяем материал
    const finalMaterialId = selectedGroupId 
      ? (groups.find(g => g.id === selectedGroupId)?.material_id || "") 
      : selectedMaterialId;

    const reqData = {
      group_id: selectedGroupId || "",
      sample: {
        sample_number: sampleNumber,
        collection_date: undefined,
        context_params: {
          "location": samplePlace, // Сохраняем место отбора в контекст
          "material_id": finalMaterialId
        },
        note: sampleNote || undefined
      },
      lab_name: labName,
      operator_name: operator, // Обратите внимание: operator_name вместо operator
      results: resultsInput
    };

    try {
      setLoading(true);
      await CreateProtocol(reqData);
      showToast("Протокол сохранен!");
      
      // Сброс формы
      setSimpleValue("");
      setRawInputs({});
      setSelectedMethodId("");
      setSampleNumber(""); 
      setSamplePlace("");
      setSampleNote("");
      
      loadProtocolsList(1);
    } catch (e: any) { 
      console.error("Ошибка сохранения:", e);
      showToast("Ошибка сохранения: " + e.toString(), true); 
    } finally {
      setLoading(false);
    }
  };

  const handleDownloadProtocolPDF = async (id: string) => {
    try {
      // 1. Получаем байты PDF
      const pdfBytes = await GenerateProtocolPDF(id);
      
      // 2. Открываем диалог сохранения
      const filePath = await SaveDialog({
        title: "Сохранить протокол",
        defaultFilename: `Protocol_${id}.pdf`,
        filters: [{ name: 'PDF', pattern: '*.pdf' }]
      });
      
      if (filePath) {
        // Wails SaveDialog возвращает путь, но не пишет файл.
        // Пишем файл через Blob в браузере (это скачает его в Загрузки, 
        // а имя файла возьмется из диалога, если браузер поддержит, или будет дефолтное)
        // Для полной интеграции лучше добавить метод SaveFile в Go, но пока так:
        const blob = new Blob([new Uint8Array(pdfBytes)], { type: 'application/pdf' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        // Пытаемся использовать имя из пути
        const fileName = filePath.split(/[/\\]/).pop() || 'protocol.pdf';
        a.download = fileName;
        a.click();
        URL.revokeObjectURL(url);
        showToast(`Файл сохранен: ${fileName}`);
      } else {
        showToast("Сохранение отменено");
      }
    } catch (e: any) { 
      console.error(e);
      showToast("Ошибка: " + e.toString(), true); 
    }
  };

  const handleDownloadGroupPDF = async (id: string) => {
    try {
      const pdfBytes = await GenerateGroupSummaryPDF(id);
      const filePath = await SaveDialog({
        title: "Сохранить сводный отчет",
        defaultFilename: `GroupSummary_${id}.pdf`,
        filters: [{ name: 'PDF', pattern: '*.pdf' }]
      });

      if (filePath) {
        const blob = new Blob([new Uint8Array(pdfBytes)], { type: 'application/pdf' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = filePath.split(/[/\\]/).pop() || 'summary.pdf';
        a.click();
        URL.revokeObjectURL(url);
        showToast(`Файл сохранен`);
      }
    } catch (e: any) { 
      showToast("Ошибка: " + e.toString(), true); 
    }
  };

  const showToast = (msg: string, error: boolean = false) => {
    setToast({ msg, error });
    setTimeout(() => setToast(null), 3000);
  };

  // === HELPERS FOR UI ===
  const currentMethod = methods.find(m => m.id === selectedMethodId);

  const getComplianceStatus = (): 'success' | 'danger' | 'neutral' => {
    if (!currentMethod) return 'neutral';
    
    // Упрощенная проверка: просто проверяем заполненность
    // Реальную валидацию по нормам делает бэкенд при сохранении, 
    // так как нормы зависят от контекста (климат, марка и т.д.)
    if (currentMethod.is_calculated) {
      const inputParams = currentMethod.parameters.filter((p: any) => p.is_input);
      const allFilled = inputParams.every((p: any) => {
        if (!p.is_required) return true;
        const val = rawInputs[p.param_key];
        return val && val.trim() !== '' && !isNaN(parseFloat(val));
      });
      return allFilled ? 'success' : 'neutral';
    } else {
      const val = parseFloat(simpleValue);
      return isNaN(val) ? 'neutral' : 'success';
    }
  };

  const statusBadge = getComplianceStatus();

  return (
    <div style={styles.container}>
      {/* HEADER */}
      <header style={styles.header}>
        <div style={styles.wrap}>
          <div style={styles.brand}>
            <div style={styles.logo}>ГОСТ</div>
            <div>
              <div style={{fontWeight: 600}}>Лаборатория дорожных материалов</div>
              <div style={styles.muted}>Система управления протоколами (v2.0)</div>
            </div>
          </div>
          <div style={styles.row}>
            <span style={{...styles.badge, ...(statusBadge === 'success' ? styles.bGreen : statusBadge === 'danger' ? styles.bRed : {} )}}>
              {statusBadge === 'success' ? 'Готов к сохранению' : statusBadge === 'danger' ? 'Не соответствует' : 'Готов к вводу'}
            </span>
          </div>
        </div>
      </header>

      {/* MAIN CONTENT */}
      <main style={{...styles.wrap, padding: '24px', overflowY: 'auto'}}>
        <div style={styles.gridMain}>
          
          {/* LEFT COLUMN */}
          <section style={styles.card}>
            <h3 style={styles.title}>Лаборатория</h3>
            <label style={styles.label}><div style={styles.lab}>Организация</div><input style={styles.input} value={labName} onChange={e => setLabName(e.target.value)} /></label>
            <label style={styles.label}><div style={styles.lab}>Оператор</div><input style={styles.input} value={operator} onChange={e => setOperator(e.target.value)} /></label>
            <label style={styles.label}><div style={styles.lab}>Проект</div><input style={styles.input} value={project} onChange={e => setProject(e.target.value)} /></label>
            
            <h3 style={{...styles.title, marginTop: 16}}>Проба</h3>
            <label style={styles.label}><div style={styles.lab}>№ пробы *</div><input style={styles.input} value={sampleNumber} onChange={e => setSampleNumber(e.target.value)} required /></label>
            <label style={styles.label}><div style={styles.lab}>Место отбора *</div><input style={styles.input} value={samplePlace} onChange={e => setSamplePlace(e.target.value)} required /></label>
            <label style={styles.label}><div style={styles.lab}>Примечание</div><textarea style={styles.input} rows={2} value={sampleNote} onChange={e => setSampleNote(e.target.value)} /></label>
            
            <h3 style={{...styles.title, marginTop: 16}}>Комплексное испытание</h3>
            <label style={styles.label}>
              <div style={styles.lab}>Группа испытаний</div>
              <select style={styles.input} value={selectedGroupId} onChange={e => {
                const gid = e.target.value;
                setSelectedGroupId(gid);
                if (gid) {
                  const g = groups.find(grp => grp.id === gid);
                  if (g) {
                    setSelectedMaterialId(g.material_id);
                    loadStandards(g.material_id);
                  }
                } else {
                  setSelectedMaterialId("");
                  setStandards([]);
                  setMethods([]);
                }
              }}>
                <option value="">— без группы —</option>
                {groups.map(g => <option key={g.id} value={g.id}>{g.name}</option>)}
              </select>
            </label>
            <button style={styles.btn} onClick={() => setIsGroupModalOpen(true)}>Создать новую группу</button>
          </section>

          {/* RIGHT COLUMN */}
          <section style={styles.card}>
            <h3 style={styles.title}>Стандарт / Метод</h3>
            <label style={styles.label}>
              <div style={styles.lab}>Материал *</div>
              <select style={styles.input}
                value={selectedMaterialId}
                disabled={!!selectedGroupId}
                onChange={e => {
                  const val = e.target.value;
                  setSelectedMaterialId(val);
                  loadStandards(val);
                }}>
                <option value="">— выберите —</option>
                {materials.map(m => <option key={m.id} value={m.id}>{m.name}</option>)}
              </select>
            </label>
            <label style={styles.label}>
              <div style={styles.lab}>Стандарт (ГОСТ) *</div>
              <select style={styles.input} value={selectedStandardId} onChange={e => { setSelectedStandardId(e.target.value); loadMethods(e.target.value); }}>
                <option value="">— выберите —</option>
                {standards.map(s => <option key={s.id} value={s.id}>{s.name}</option>)}
              </select>
            </label>
            <label style={styles.label}>
              <div style={styles.lab}>Метод *</div>
              <select style={styles.input} value={selectedMethodId} onChange={e => setSelectedMethodId(e.target.value)}>
                <option value="">— выберите —</option>
                {methods.map(m => <option key={m.id} value={m.id}>{m.name} ({m.unit})</option>)}
              </select>
            </label>

            <h3 style={{...styles.title, marginTop: 16}}>Результат</h3>
            <div style={{marginBottom: 16}}>
              {!currentMethod ? (
                <div style={styles.muted}>Выберите метод для ввода.</div>
              ) : currentMethod.is_calculated ? (
                <div>
                  <div style={styles.lab}>{currentMethod.name} ({currentMethod.unit}) - Расчётный</div>
                  {currentMethod.parameters.filter((p: any) => p.is_input).map((p: any) => (
                    <label key={p.param_key} style={styles.label}>
                      <div style={styles.lab}>{p.label} ({p.unit}) {p.is_required && '*'}</div>
                      <input style={styles.input} type="number" step="any"
                        value={rawInputs[p.param_key] || ''}
                        onChange={e => setRawInputs({...rawInputs, [p.param_key]: e.target.value})}
                      />
                    </label>
                  ))}
                  <div style={{marginTop: 8, fontSize: 12, color: '#666'}}>Формула: {currentMethod.formula}</div>
                </div>
              ) : (
                <label style={styles.label}>
                  <div style={styles.lab}>{currentMethod.name} ({currentMethod.unit})</div>
                  <input style={styles.input} type="number" step="any" value={simpleValue} onChange={e => setSimpleValue(e.target.value)} />
                </label>
              )}
            </div>
            <div style={styles.actions}>
              <button style={styles.btn} onClick={() => { setSimpleValue(''); setRawInputs({}); setSelectedMethodId(''); }}>Сброс</button>
            </div>
          </section>
        </div>

        {/* COMPARISON & SAVE */}
        <section style={{...styles.card, marginTop: 24}}>
          <div style={styles.row}>
            <h3 style={styles.title}>Сохранение протокола</h3>
            <button style={{...styles.btn, background: loading ? '#ccc' : '#16a34a', color: 'white'}} 
              disabled={loading} 
              onClick={handleSaveProtocol}>
              {loading ? 'Сохранение...' : 'Сохранить протокол'}
            </button>
          </div>
          <div style={{marginTop: 12, padding: 12, background: statusBadge === 'success' ? '#dcfce7' : statusBadge === 'danger' ? '#fee2e2' : '#f1f5f9', borderRadius: 8}}>
            {currentMethod ? (
              statusBadge === 'success' ? <span style={{color: '#166534'}}>✓ Данные заполнены корректно</span> :
              statusBadge === 'danger' ? <span style={{color: '#991b1b'}}>✗ Ошибка ввода</span> :
              <span style={{color: '#64748b'}}>Ожидание ввода данных...</span>
            ) : <span style={{color: '#64748b'}}>Выберите метод</span>}
          </div>
        </section>

        {/* HISTORY GRIDS */}
        <div style={styles.gridHistory}>
          {/* PROTOCOLS */}
          <section style={styles.card}>
            <h3 style={styles.title}>История протоколов</h3>
            <div style={{overflowX: 'auto'}}>
              <table style={styles.table}>
                <thead>
                  <tr><th style={styles.th}>Дата</th><th style={styles.th}>№</th><th style={styles.th}>Материал</th><th style={styles.th}>Лаборатория</th><th style={styles.th}>Действия</th></tr>
                </thead>
                <tbody>
                  {protocols.length === 0 ? (
                    <tr><td colSpan={5} style={{textAlign:'center', padding: 20}}>Нет протоколов</td></tr>
                  ) : (
                    protocols.map((p) => (
                      <tr key={p.id} style={styles.tr} onClick={() => setViewingProtocol(p)}>
                        <td style={styles.td}>{p.created_at ? new Date(p.created_at).toLocaleDateString() : '-'}</td>
                        <td style={styles.td}>{p.sample_number || '-'}</td>
                        <td style={styles.td}>{p.material_name || '-'}</td>
                        <td style={styles.td}>{p.lab_name}</td>
                        <td style={styles.td}>
                          <button style={styles.smallBtn} onClick={(e) => { e.stopPropagation(); handleDownloadProtocolPDF(p.id); }}>PDF</button>
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
            {protocolsMeta && protocolsMeta.total_pages > 1 && (
              <div style={styles.pagination}>
                <button style={styles.btn} disabled={protocolsPage===1} onClick={()=>loadProtocolsList(protocolsPage-1)}>←</button>
                <span>{protocolsPage} / {protocolsMeta.total_pages}</span>
                <button style={styles.btn} disabled={protocolsPage===protocolsMeta.total_pages} onClick={()=>loadProtocolsList(protocolsPage+1)}>→</button>
              </div>
            )}
          </section>

          {/* GROUPS */}
          <section style={styles.card}>
            <h3 style={styles.title}>Комплексные испытания</h3>
            <div style={{overflowX: 'auto'}}>
              <table style={styles.table}>
                <thead>
                  <tr><th style={styles.th}>Название</th><th style={styles.th}>Материал</th><th style={styles.th}>Проект</th><th style={styles.th}>Действия</th></tr>
                </thead>
                <tbody>
                  {groups.length === 0 ? (
                    <tr><td colSpan={4} style={{textAlign:'center', padding: 20}}>Нет групп</td></tr>
                  ) : (
                    groups.map(g => (
                      <tr key={g.id} style={styles.tr}>
                        <td style={styles.td}>{g.name}</td>
                        <td style={styles.td}>{materials.find(m => m.id === g.material_id)?.name || '-'}</td>
                        <td style={styles.td}>{g.project_name || '-'}</td>
                        <td style={styles.td}>
                          <button style={styles.smallBtn} onClick={async () => {
                            try {
                              const sum = await GetGroupSummary(g.id);
                              setViewingSummary(sum);
                            } catch(e) { showToast("Ошибка загрузки сводки", true); }
                          }}>Сводка</button>
                          <button style={{...styles.smallBtn, marginLeft: 4}} onClick={() => handleDownloadGroupPDF(g.id)}>PDF</button>
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
            {groupsMeta && groupsMeta.total_pages > 1 && (
              <div style={styles.pagination}>
                <button style={styles.btn} disabled={groupsPage===1} onClick={()=>loadGroupsList(groupsPage-1)}>←</button>
                <span>{groupsPage} / {groupsMeta.total_pages}</span>
                <button style={styles.btn} disabled={groupsPage===groupsMeta.total_pages} onClick={()=>loadGroupsList(groupsPage+1)}>→</button>
              </div>
            )}
          </section>
        </div>
      </main>

      {/* MODALS */}
      {isGroupModalOpen && (
        <div style={styles.modalOverlay} onClick={()=>setIsGroupModalOpen(false)}>
          <div style={styles.modal} onClick={e=>e.stopPropagation()}>
            <h3 style={styles.title}>Новая группа</h3>
            <label style={styles.label}><div style={styles.lab}>Название *</div><input style={styles.input} value={newGroupName} onChange={e=>setNewGroupName(e.target.value)} /></label>
            <label style={styles.label}><div style={styles.lab}>Проект</div><input style={styles.input} value={newGroupDesc} onChange={e=>setNewGroupDesc(e.target.value)} /></label>
            <label style={styles.label}><div style={styles.lab}>Материал *</div>
              <select style={styles.input} value={newGroupMatId} onChange={e=>setNewGroupMatId(e.target.value)}>
                <option value="">— выберите —</option>
                {materials.map(m => <option key={m.id} value={m.id}>{m.name}</option>)}
              </select>
            </label>
            <div style={styles.actions}>
              <button style={styles.btn} onClick={()=>setIsGroupModalOpen(false)}>Отмена</button>
              <button style={{...styles.btn, background:'#0f172a', color:'white'}} onClick={handleCreateGroup}>Создать</button>
            </div>
          </div>
        </div>
      )}

      {viewingProtocol && (
        <div style={styles.modalOverlay} onClick={()=>setViewingProtocol(null)}>
          <div style={{...styles.modal, maxWidth: 800}} onClick={e=>e.stopPropagation()}>
            <div style={styles.row}>
              <h3 style={styles.title}>Протокол #{viewingProtocol.sample_number}</h3>
              <button style={styles.btn} onClick={()=>setViewingProtocol(null)}>✕</button>
            </div>
            <div style={{marginBottom: 16, fontSize: 13, background: '#f8fafc', padding: 12, borderRadius: 8}}>
              <strong>Дата:</strong> {viewingProtocol.created_at ? new Date(viewingProtocol.created_at).toLocaleString() : '-'}<br/>
              <strong>Материал:</strong> {viewingProtocol.material_name}<br/>
              <strong>Лаборатория:</strong> {viewingProtocol.lab_name}
            </div>
            <div style={{...styles.actions, justifyContent: 'flex-end'}}>
               <button style={{...styles.btn, background:'#0f172a', color:'white'}} onClick={() => handleDownloadProtocolPDF(viewingProtocol.id)}>Скачать PDF</button>
            </div>
          </div>
        </div>
      )}

      {viewingSummary && (
        <div style={styles.modalOverlay} onClick={()=>setViewingSummary(null)}>
          <div style={{...styles.modal, maxWidth: 900, maxHeight: '90vh', overflowY: 'auto'}} onClick={e=>e.stopPropagation()}>
            <div style={styles.row}>
              <h3 style={styles.title}>Сводка: {viewingSummary.group_name}</h3>
              <button style={styles.btn} onClick={()=>setViewingSummary(null)}>✕</button>
            </div>
            <div style={{marginBottom: 16, display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12, background: '#f8fafc', padding: 12, borderRadius: 8}}>
              <div><strong>Материал:</strong> {materials.find(m=>m.id===viewingSummary.material_id)?.name}</div>
              <div><strong>Проб:</strong> {viewingSummary.total_samples}</div>
              <div><strong>Соответствие:</strong> {viewingSummary.compliant_rate.toFixed(1)}%</div>
            </div>
            
            <table style={styles.table}>
              <thead>
                <tr>
                  <th style={styles.th}>Метод</th>
                  <th style={styles.th}>Норма</th>
                  <th style={styles.th}>Статус</th>
                  <th style={styles.th}>Пробы ({viewingSummary.total_samples})</th>
                </tr>
              </thead>
              <tbody>
                {viewingSummary.results.map((r, i) => {
                  const norm = (r.min_value != null && r.max_value != null) ? `${r.min_value} – ${r.max_value}` : 
                               (r.min_value != null) ? `≥${r.min_value}` : 
                               (r.max_value != null) ? `≤${r.max_value}` : '—';
                  
                  return (
                    <tr key={i} style={styles.tr}>
                      <td style={styles.td}><strong>{r.method_name}</strong><br/><span style={{fontSize:11, color:'#666'}}>{r.unit}</span></td>
                      <td style={styles.td}>{norm}</td>
                      <td style={styles.td}>
                        {r.is_compliant ? <span style={styles.bGreen}>✓ Соответствует</span> : <span style={styles.bRed}>✗ Есть отклонения</span>}
                      </td>
                      <td style={styles.td}>
                        <div style={{display: 'flex', flexWrap: 'wrap', gap: 4}}>
                          {r.trials.map((t, idx) => (
                            <span key={idx} title={`Проба ${t.sample_number}: ${t.value}`} 
                              style={{
                                fontSize: 11, padding: '2px 6px', borderRadius: 4, 
                                background: t.is_compliant ? '#dcfce7' : '#fee2e2',
                                color: t.is_compliant ? '#166534' : '#991b1b'
                              }}>
                              {t.sample_number}: {t.value}
                            </span>
                          ))}
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
            
            <div style={{...styles.actions, marginTop: 16, justifyContent: 'flex-end'}}>
               <button style={{...styles.btn, background:'#0f172a', color:'white'}} onClick={() => handleDownloadGroupPDF(viewingSummary.group_id)}>Скачать PDF</button>
            </div>
          </div>
        </div>
      )}

      {/* TOAST */}
      {toast && (
        <div style={{...styles.toast, borderLeft: `4px solid ${toast.error ? 'var(--red)' : 'var(--green)'}`}}>
          {toast.msg}
        </div>
      )}
    </div>
  );
}

// === STYLES (Оставлены без изменений из старого проекта) ===
const styles: Record<string, any> = {
  container: { display: 'flex', flexDirection: 'column', height: '100vh', background: '#f1f5f9', color: '#0f172a', fontFamily: 'system-ui, -apple-system, sans-serif' },
  header: { position: 'sticky', top: 0, zIndex: 10, background: 'rgba(255,255,255,0.9)', backdropFilter: 'blur(6px)', borderBottom: '1px solid #e2e8f0' },
  wrap: { maxWidth: 1400, margin: '0 auto', padding: '16px 24px', width: '100%', boxSizing: 'border-box' },
  brand: { display: 'flex', alignItems: 'center', gap: 12 },
  logo: { width: 36, height: 36, borderRadius: 12, background: '#0f172a', color: '#fff', display: 'grid', placeItems: 'center', fontWeight: 700, fontSize: 12 },
  muted: { color: '#64748b', fontSize: 12 },
  row: { display: 'flex', gap: 8, alignItems: 'center', justifyContent: 'space-between' },
  badge: { display: 'inline-block', fontSize: 12, padding: '4px 10px', borderRadius: 999, background: '#e2e8f0', color: '#0f172a', fontWeight: 500 },
  bGreen: { background: '#dcfce7', color: '#166534', padding: '2px 6px', borderRadius: 4, fontSize: 12 },
  bRed: { background: '#fee2e2', color: '#991b1b', padding: '2px 6px', borderRadius: 4, fontSize: 12 },
  gridMain: { display: 'grid', gap: 24, gridTemplateColumns: 'repeat(auto-fit, minmax(300px, 1fr))' },
  gridHistory: { display: 'grid', gap: 24, gridTemplateColumns: 'repeat(auto-fit, minmax(400px, 1fr))', marginTop: 24 },
  card: { background: '#fff', border: '1px solid #e2e8f0', borderRadius: 16, padding: 20, boxShadow: '0 1px 3px rgba(0,0,0,0.05)' },
  title: { fontWeight: 600, margin: '0 0 16px', fontSize: 16, color: '#1e293b' },
  label: { display: 'block', margin: '10px 0' },
  lab: { fontSize: 12, color: '#64748b', marginBottom: 4, fontWeight: 500 },
  input: { width: '100%', boxSizing: 'border-box', padding: '8px 12px', borderRadius: 8, border: '1px solid #cbd5e1', background: '#fff', font: 'inherit', transition: 'border-color 0.2s' },
  actions: { display: 'flex', gap: 8, marginTop: 16 },
  btn: { padding: '8px 16px', border: '1px solid #cbd5e1', background: '#fff', borderRadius: 8, cursor: 'pointer', font: 'inherit', fontWeight: 500, transition: 'all 0.2s' },
  smallBtn: { padding: '4px 8px', fontSize: 11, background: '#e2e8f0', border: 'none', borderRadius: 4, cursor: 'pointer', fontWeight: 600 },
  table: { borderCollapse: 'collapse', width: '100%', fontSize: 13 },
  th: { borderBottom: '2px solid #e2e8f0', padding: 10, textAlign: 'left', fontWeight: 600, color: '#475569' },
  td: { borderTop: '1px solid #f1f5f9', padding: 10, textAlign: 'left', verticalAlign: 'top' },
  tr: { cursor: 'pointer', transition: 'background 0.1s' },
  pagination: { display: 'flex', gap: 8, marginTop: 16, justifyContent: 'center', alignItems: 'center', fontSize: 13 },
  modalOverlay: { position: 'fixed', top: 0, left: 0, right: 0, bottom: 0, background: 'rgba(15, 23, 42, 0.6)', zIndex: 100, display: 'grid', placeItems: 'center', backdropFilter: 'blur(2px)' },
  modal: { background: '#fff', padding: 24, borderRadius: 16, width: '90%', maxWidth: 500, boxShadow: '0 20px 25px -5px rgba(0, 0, 0, 0.1)', animation: 'fadeIn 0.2s ease-out' },
  toast: { position: 'fixed', bottom: 24, left: '50%', transform: 'translateX(-50%)', background: '#fff', border: '1px solid #e2e8f0', padding: '12px 24px', borderRadius: 12, boxShadow: '0 10px 15px -3px rgba(0, 0, 0, 0.1)', zIndex: 1000, fontWeight: 500, minWidth: 200, textAlign: 'center' }
};