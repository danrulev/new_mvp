// src/pages/LabDashboard.tsx
import { useState, useEffect, useMemo } from 'react';
import { models } from '../../wailsjs/go/models';
import {
  CreateGroup,
  CreateProtocolWithSample,
  GetGroups,
  GetMaterials,
  GetMethodDetails,
  GetMethodsByStandardID,
  GetProtocolFull, // 🔥 Импортируем GetProtocolFull
  GetProtocols,
  GetStandardsByMaterialID,
  SaveProtocolPDFWithDialog,
} from '../../wailsjs/go/app/App';

// ============================================================================
// TYPE ALIASES
// ============================================================================
type Material = models.Material;
type Standard = models.Standard;
type TestMethod = models.TestMethod;
type MethodInput = models.MethodInputDTO;
type ExperimentGroup = models.ExperimentGroup;
type Protocol = models.Protocol;
type TestResult = models.TestResult;
type GroupListResponse = models.GroupListResponse;
type ProtocolListResponse = models.ProtocolListResponse;
type CreateProtocolRequest = models.CreateProtocolRequest;
type CreateSampleDTO = models.CreateSampleDTO;
type CreateResultDTO = models.CreateResultDTO;
type ProtocolFull = models.ProtocolFull;
type TestMethodFull = models.TestMethodFull;
type NormativeLimit = models.NormativeLimit;

interface TestMethodWithInputs extends TestMethod {
  inputs?: MethodInput[];
  limits?: any[];
}

// Интерфейс для строки таблицы с рассчитанной нормой
interface ResultRow {
    id: string;
    protocol_id: string;
    method_id: string;
    input_data: Record<string, any>;
    calculated_value?: number;
    applied_limit_id?: string;
    is_compliant?: boolean;
    deviation_msg?: string;
    note?: string;
    created_at: any;
    
    // Наши дополнительные поля
    methodName: string;
    normString: string;
}

// ============================================================================
// ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ
// ============================================================================
const safeCall = async <T,>(
  fn: () => Promise<T>,
  errorMsg: string,
  onError?: (e: any) => void
): Promise<T | null> => {
  try {
    return await fn();
  } catch (e) {
    console.error(`${errorMsg}:`, e);
    onError?.(e);
    return null;
  }
};

const formatDate = (date: any): string => {
  if (!date) return '—';
  try {
    const d = new Date(date);
    return d.toLocaleDateString('ru-RU', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  } catch {
    return String(date);
  }
};

const formatNumber = (value: number | undefined | null, unit: string = ''): string => {
  if (value === undefined || value === null || isNaN(value as number)) return '—';
  return `${(value as number).toFixed(2)}${unit ? ` ${unit}` : ''}`;
};

// Функция для форматирования строки нормы
const getNormString = (limitType?: string, min?: number, max?: number): string => {
  if (!limitType) return '—';
  if (limitType === 'range' && min !== undefined && max !== undefined) {
    return `${min.toFixed(2)} – ${max.toFixed(2)}`;
  }
  if (limitType === 'min' && min !== undefined) {
    return `≥ ${min.toFixed(2)}`;
  }
  if (limitType === 'max' && max !== undefined) {
    return `≤ ${max.toFixed(2)}`;
  }
  return '—';
};

// ============================================================================
// UI КОМПОНЕНТЫ
// ============================================================================
const Toast: React.FC<{ message: string; error?: boolean; onClose: () => void }> = ({
  message,
  error = false,
  onClose,
}) => (
  <div
    style={{
      position: 'fixed',
      bottom: 24,
      left: '50%',
      transform: 'translateX(-50%)',
      background: error ? '#fef2f2' : '#f0fdf4',
      border: `1px solid ${error ? '#fecaca' : '#bbf7d0'}`,
      borderLeft: `4px solid ${error ? '#ef4444' : '#22c55e'}`,
      padding: '12px 20px',
      borderRadius: 8,
      boxShadow: '0 4px 6px rgba(0, 0, 0, 0.1)',
      zIndex: 1000,
      display: 'flex',
      alignItems: 'center',
      gap: 12,
      fontSize: 14,
      color: error ? '#991b1b' : '#166534',
      minWidth: 300,
    }}
  >
    <span style={{ fontWeight: 600 }}>{error ? 'Ошибка' : 'Успешно'}</span>
    <span style={{ flex: 1 }}>{message}</span>
    <button onClick={onClose} style={{ background: 'none', border: 'none', fontSize: 20, cursor: 'pointer', color: 'inherit', opacity: 0.6 }}>×</button>
  </div>
);

const Modal: React.FC<{ isOpen: boolean; onClose: () => void; title: string; children: React.ReactNode }> = ({
  isOpen,
  onClose,
  title,
  children,
}) => {
  if (!isOpen) return null;
  return (
    <div
      style={{
        position: 'fixed',
        top: 0, left: 0, right: 0, bottom: 0,
        background: 'rgba(0, 0, 0, 0.5)',
        zIndex: 100,
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
      }}
      onClick={onClose}
    >
      <div
        style={{
          background: '#fff',
          padding: 24,
          borderRadius: 8,
          width: '90%',
          maxWidth: 900,
          boxShadow: '0 20px 25px rgba(0, 0, 0, 0.1)',
          maxHeight: '90vh',
          overflowY: 'auto',
        }}
        onClick={(e) => e.stopPropagation()}
      >
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 20, paddingBottom: 12, borderBottom: '1px solid #e5e7eb' }}>
          <h3 style={{ margin: 0, fontSize: 18, fontWeight: 600 }}>{title}</h3>
          <button onClick={onClose} style={{ background: 'none', border: 'none', fontSize: 24, cursor: 'pointer', color: '#6b7280' }}>×</button>
        </div>
        {children}
      </div>
    </div>
  );
};

// ============================================================================
// ОСНОВНОЙ КОМПОНЕНТ
// ============================================================================
export default function LabDashboard() {
  // === STATE: Справочники ===
  const [materials, setMaterials] = useState<Material[]>([]);
  const [standards, setStandards] = useState<Standard[]>([]);
  const [methods, setMethods] = useState<TestMethodWithInputs[]>([]);
  
  // === STATE: Форма ===
  const [labName, setLabName] = useState('БГТУ им. В.Г. Шухова');
  const [operator, setOperator] = useState('');
  const [sampleNumber, setSampleNumber] = useState('');
  const [samplePlace, setSamplePlace] = useState('');
  const [sampleNote, setSampleNote] = useState('');
  const [selectedGroupId, setSelectedGroupId] = useState<string>('');
  const [selectedMaterialId, setSelectedMaterialId] = useState<string>('');
  const [selectedStandardId, setSelectedStandardId] = useState<string>('');
  const [selectedMethodId, setSelectedMethodId] = useState<string>('');

  const [simpleValue, setSimpleValue] = useState<string>('');
  const [rawInputs, setRawInputs] = useState<Record<string, string>>({});

  // === STATE: Списки ===
  const [groups, setGroups] = useState<ExperimentGroup[]>([]);
  const [protocols, setProtocols] = useState<Protocol[]>([]);
  const [protocolsPage, setProtocolsPage] = useState(1);

  // === STATE: UI ===
  const [loading, setLoading] = useState<Record<string, boolean>>({
    materials: true, groups: true, protocols: true, standards: false, methods: false, save: false,
  });
  const [isGroupModalOpen, setIsGroupModalOpen] = useState(false);
  const [newGroup, setNewGroup] = useState({ name: '', materialId: '', project: '', location: '' });
  
  // Храним полные данные протокола для просмотра
  const [viewingData, setViewingData] = useState<ProtocolFull | null>(null);
  // Храним подготовленные строки результатов с нормами
  const [viewingResults, setViewingResults] = useState<ResultRow[]>([]);
  
  const [toast, setToast] = useState<{ msg: string; error: boolean } | null>(null);

  // === INIT ===
  useEffect(() => {
    loadMaterials();
    loadGroupsList(1);
    loadProtocolsList(1);
  }, []);

  // === LOADERS ===
  const loadMaterials = async () => {
    setLoading((p) => ({ ...p, materials: true }));
    const list = await safeCall(() => GetMaterials(), 'Ошибка загрузки материалов');
    if (list) setMaterials(list);
    setLoading((p) => ({ ...p, materials: false }));
  };

  const loadGroupsList = async (page: number) => {
    setLoading((p) => ({ ...p, groups: true }));
    const offset = (page - 1) * 10;
    const response = await safeCall(() => GetGroups(10, offset) as Promise<GroupListResponse>, 'Ошибка загрузки групп');
    if (response) {
      setGroups(response.items || []);
      setProtocolsPage(page);
    }
    setLoading((p) => ({ ...p, groups: false }));
  };

  const loadProtocolsList = async (page: number) => {
    setLoading((p) => ({ ...p, protocols: true }));
    const offset = (page - 1) * 10;
    const response = await safeCall(() => GetProtocols(10, offset) as Promise<ProtocolListResponse>, 'Ошибка загрузки протоколов');
    if (response) {
      setProtocols(response.items || []);
      setProtocolsPage(page);
    }
    setLoading((p) => ({ ...p, protocols: false }));
  };

  const loadStandards = async (materialId: string) => {
    if (!materialId) { setStandards([]); setMethods([]); return; }
    setLoading((p) => ({ ...p, standards: true }));
    try {
      const list = await GetStandardsByMaterialID(materialId);
      if (list) {
        setStandards(list);
        setMethods([]);
        setSelectedStandardId('');
        setSelectedMethodId('');
        setRawInputs({});
        setSimpleValue('');
      }
    } catch (e) { console.error(e); }
    finally { setLoading((p) => ({ ...p, standards: false })); }
  };

  const loadMethods = async (standardId: string) => {
    if (!standardId) { setMethods([]); return; }
    setLoading((p) => ({ ...p, methods: true }));
    const list = await safeCall(() => GetMethodsByStandardID(standardId) as Promise<TestMethod[]>, 'Ошибка загрузки методов');
    if (list) {
      setMethods(list as TestMethodWithInputs[]);
      setSelectedMethodId('');
      setRawInputs({});
      setSimpleValue('');
    }
    setLoading((p) => ({ ...p, methods: false }));
  };

  const loadMethodDetails = async (methodId: string) => {
    if (!methodId) return;
    setSelectedMethodId(methodId);
    setRawInputs({});
    setSimpleValue('');
    const details = await safeCall(() => GetMethodDetails(methodId), 'Ошибка загрузки деталей метода');
    if (details) {
      const extended: TestMethodWithInputs = {
        id: details.method.id,
        standard_id: details.method.standard_id,
        name: details.method.name,
        unit: details.method.unit,
        result_type: details.method.result_type,
        is_mandatory: details.method.is_mandatory,
        code: details.method.code,
        description: details.method.description,
        formula_expr: details.method.formula_expr,
        inputs: details.inputs ?? [],
        limits: details.limits,
      };
      setMethods((prev) => prev.map((m) => (m.id === methodId ? { ...m, ...extended } : m)));
    }
  };

  // === COMPUTED ===
  const currentMethod = useMemo(() => methods.find((m) => m.id === selectedMethodId), [methods, selectedMethodId]);
  const isCalculated = useMemo(() => !!(currentMethod?.inputs && currentMethod.inputs.length > 0), [currentMethod]);
  
  const allInputsFilled = useMemo(() => {
    if (!currentMethod) return false;
    if (isCalculated) {
      return currentMethod.inputs?.every((inp: MethodInput) => {
        if (!inp.is_required) return true;
        const val = rawInputs[inp.param_key];
        return val && val.trim() !== '';
      }) ?? false;
    } else {
      const val = parseFloat(simpleValue);
      return !isNaN(val);
    }
  }, [currentMethod, isCalculated, rawInputs, simpleValue]);

  const canSave = !!(
    sampleNumber && samplePlace && selectedMaterialId && selectedMethodId && 
    allInputsFilled && labName.trim() !== '' && operator.trim() !== ''
  );

  const calculatedPreview = useMemo(() => {
    if (!currentMethod || !isCalculated || !currentMethod.formula_expr) return undefined;
    const inputs: Record<string, number> = {};
    for (const inp of currentMethod.inputs ?? []) {
      if (rawInputs[inp.param_key]) {
        const val = parseFloat(rawInputs[inp.param_key]);
        if (!isNaN(val)) inputs[inp.param_key] = val;
      }
    }
    try {
      let expr = currentMethod.formula_expr;
      for (const [key, val] of Object.entries(inputs)) {
        expr = expr.replace(new RegExp(`\\b${key}\\b`, 'g'), val.toString());
      }
      if (/^[\d\s\+\-\*\/\.\(\)]+$/.test(expr)) {
        const result = Function('"use strict"; return (' + expr + ')')();
        return Math.round(result * 100) / 100;
      }
    } catch {}
    return undefined;
  }, [currentMethod, isCalculated, rawInputs]);

  // === HANDLERS ===
  const showToast = (msg: string, error = false) => {
    setToast({ msg, error });
    setTimeout(() => setToast(null), error ? 5000 : 3000);
  };

  const handleCreateGroup = async () => {
    if (!newGroup.name || !newGroup.materialId) { showToast('Заполните название и материал', true); return; }
    setLoading((p) => ({ ...p, save: true }));
    const result = await safeCall(() => CreateGroup(newGroup.name, newGroup.project, newGroup.location || '', newGroup.materialId), 'Ошибка создания группы');
    setLoading((p) => ({ ...p, save: false }));
    if (result) {
      showToast('Группа создана');
      setIsGroupModalOpen(false);
      setNewGroup({ name: '', materialId: '', project: '', location: '' });
      loadGroupsList(1);
    }
  };

  const handleSaveProtocol = async () => {
    if (!canSave) { 
      if (!labName.trim()) { showToast('Укажите лабораторию', true); return; }
      if (!operator.trim()) { showToast('Укажите оператора', true); return; }
      showToast('Заполните все поля', true); 
      return; 
    }

    const method = methods.find((m) => m.id === selectedMethodId);
    if (!method) return;

    let resultsInput: CreateResultDTO[] = [];
    if (isCalculated) {
      for (const p of currentMethod?.inputs ?? []) {
        const valStr = rawInputs[p.param_key];
        if (p.is_required && (!valStr || valStr.trim() === '')) {
          showToast(`Заполните: ${p.label}`, true); return;
        }
      }
      resultsInput = [{ method_id: method.id, raw_inputs: { ...rawInputs } } as CreateResultDTO];
    } else {
      const val = parseFloat(simpleValue);
      if (isNaN(val)) { showToast('Некорректное число', true); return; }
      resultsInput = [{ method_id: method.id, raw_inputs: { value: val.toString() } } as CreateResultDTO];
    }

    const sampleData = models.CreateSampleDTO.createFrom({
      sample_number: sampleNumber,
      material_id: selectedMaterialId,
      collection_date: undefined,
      collection_place: samplePlace, 
      context_params: {},
      note: sampleNote || undefined,
    });

    const reqData = {
      group_id: selectedGroupId || '',
      sample: sampleData,
      lab_name: labName,
      operator_name: operator,
      results: resultsInput,
    } as CreateProtocolRequest;

    try {
      setLoading((p) => ({ ...p, save: true }));
      await CreateProtocolWithSample(reqData);
      showToast('Протокол сохранён');
      
      setRawInputs({}); setSimpleValue(''); setSelectedMethodId('');
      setSampleNumber(''); setSamplePlace(''); setSampleNote('');
      loadProtocolsList(1);
    } catch (e: any) {
      showToast('Ошибка: ' + (e.message || String(e)), true);
    } finally {
      setLoading((p) => ({ ...p, save: false }));
    }
  };

  const handleDownloadProtocolPDF = async (id: string) => {
    setLoading((p) => ({ ...p, save: true }));
    const path = await safeCall(() => SaveProtocolPDFWithDialog(id), 'Ошибка PDF');
    setLoading((p) => ({ ...p, save: false }));
    if (path) showToast(`Файл: ${path.split(/[/\\]/).pop()}`);
  };

  // 🔥 ОБНОВЛЕННАЯ ФУНКЦИЯ ПРОСМОТРА С РАСЧЕТОМ НОРМЫ
    const handleViewProtocol = async (id: string) => {
    setLoading((p) => ({ ...p, save: true }));
    
    // 1. Загружаем полный протокол
    const fullData = await safeCall(() => GetProtocolFull(id), 'Ошибка загрузки протокола');
    
    if (!fullData || !fullData.results || fullData.results.length === 0) {
      setViewingData(fullData);
      setViewingResults([]);
      setLoading((p) => ({ ...p, save: false }));
      return;
    }

    setViewingData(fullData);

    // 2. Пытаемся определить StandardID по первому результату
    const firstMethodId = fullData.results[0].method_id;
    
    // Загружаем детали первого метода, чтобы узнать StandardID
    const firstMethodBasic = await safeCall(() => GetMethodDetails(firstMethodId), 'Ошибка загрузки метода');

    if (firstMethodBasic && firstMethodBasic.method && firstMethodBasic.method.standard_id) {
        const standardId = firstMethodBasic.method.standard_id;

        // 3. Загружаем ВСЕ методы и лимиты этого стандарта
        // Примечание: Убедитесь, что вы добавили этот метод в app.go и сделали wails build
        // Если метода нет, норма останется прочерком.
        const methodsFullMap = await safeCall(() => 
            // @ts-ignore - если метод еще не сгенерирован, игнорируем ошибку типа временно
            window.app.GetMethodsFullByStandardID(standardId) 
        , 'Ошибка загрузки методов стандарта');

        // 4. Формируем строки таблицы
        const processedRows: ResultRow[] = fullData.results.map(res => {
            let normStr = '—';
            let methodName = res.method_id;

            // Попытка найти метод в кэше формы (если стандарт тот же)
            const cachedMethod = methods.find(m => m.id === res.method_id);
            
            if (cachedMethod) {
                methodName = cachedMethod.name;
                // Берем первый лимит для превью (упрощенно)
                if (cachedMethod.limits && cachedMethod.limits.length > 0) {
                    const limit = cachedMethod.limits[0] as any; 
                    if (limit) {
                         normStr = getNormString(limit.limit_type, limit.min_value, limit.max_value);
                    }
                }
            } else if (methodsFullMap && (methodsFullMap as any)[res.method_id]) {
                // Если загрузили полную мапу через API
                const fullM = (methodsFullMap as any)[res.method_id] as TestMethodFull;
                methodName = fullM.method.name;
                
                if (fullM.limits && fullM.limits.length > 0) {
                     const limit = fullM.limits[0];
                     normStr = getNormString(limit.limit_type, limit.min_value, limit.max_value);
                }
            }

            // ✅ Явно создаем объект, копируя нужные поля из res (класса) в наш интерфейс
            return {
                id: res.id,
                protocol_id: res.protocol_id,
                method_id: res.method_id,
                input_data: res.input_data,
                calculated_value: res.calculated_value,
                applied_limit_id: res.applied_limit_id,
                is_compliant: res.is_compliant,
                deviation_msg: res.deviation_msg,
                note: res.note,
                created_at: res.created_at,
                // Дополнительные поля
                methodName: methodName,
                normString: normStr
            };
        });

        setViewingResults(processedRows);
    } else {
        // Фоллбэк, если не удалось найти стандарт
        const fallbackRows: ResultRow[] = fullData.results.map(r => ({
            id: r.id,
            protocol_id: r.protocol_id,
            method_id: r.method_id,
            input_data: r.input_data,
            calculated_value: r.calculated_value,
            applied_limit_id: r.applied_limit_id,
            is_compliant: r.is_compliant,
            deviation_msg: r.deviation_msg,
            note: r.note,
            created_at: r.created_at,
            methodName: r.method_id,
            normString: '—'
        }));
        setViewingResults(fallbackRows);
    }

    setLoading((p) => ({ ...p, save: false }));
  };

  // ============================================================================
  // STYLES
  // ============================================================================
  const styles = {
    container: { display: 'flex', flexDirection: 'column' as const, height: '100vh', background: '#f3f4f6', color: '#1f2937', fontFamily: 'system-ui, sans-serif' },
    header: { background: '#fff', borderBottom: '1px solid #e5e7eb', padding: '16px 24px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' },
    title: { fontSize: 20, fontWeight: 700, margin: 0 },
    subtitle: { fontSize: 13, color: '#6b7280', marginTop: 4 },
    main: { flex: 1, overflowY: 'auto' as const, padding: 24, maxWidth: 1400, margin: '0 auto', width: '100%', boxSizing: 'border-box' as const },
    grid: { display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(350px, 1fr))', gap: 24 },
    card: { background: '#fff', borderRadius: 8, border: '1px solid #e5e7eb', padding: 20 },
    sectionTitle: { fontSize: 16, fontWeight: 600, margin: '0 0 16px', borderBottom: '1px solid #f3f4f6', paddingBottom: 8 },
    label: { display: 'block', marginBottom: 12 },
    labelText: { display: 'block', fontSize: 13, fontWeight: 500, color: '#374151', marginBottom: 4 },
    input: { width: '100%', padding: '8px 12px', borderRadius: 6, border: '1px solid #d1d5db', fontSize: 14, boxSizing: 'border-box' as const },
    select: { width: '100%', padding: '8px 12px', borderRadius: 6, border: '1px solid #d1d5db', fontSize: 14, background: '#fff' },
    btnPrimary: { background: '#2563eb', color: '#fff', border: 'none', padding: '10px 20px', borderRadius: 6, fontSize: 14, fontWeight: 600, cursor: 'pointer', width: '100%' },
    btnSecondary: { background: '#fff', color: '#374151', border: '1px solid #d1d5db', padding: '8px 16px', borderRadius: 6, fontSize: 13, cursor: 'pointer', marginTop: 8, width: '100%' },
    btnSmall: { background: '#f3f4f6', border: '1px solid #d1d5db', padding: '4px 8px', borderRadius: 4, fontSize: 12, cursor: 'pointer' },
    table: { width: '100%', borderCollapse: 'collapse' as const, fontSize: 13 },
    th: { textAlign: 'left' as const, padding: '10px', borderBottom: '2px solid #e5e7eb', color: '#6b7280', fontWeight: 600 },
    td: { padding: '10px', borderBottom: '1px solid #f3f4f6' },
    badge: { display: 'inline-block', padding: '2px 8px', borderRadius: 999, fontSize: 11, fontWeight: 600 },
    badgeSuccess: { background: '#dcfce7', color: '#166534' },
    badgeDanger: { background: '#fee2e2', color: '#991b1b' },
    badgeNeutral: { background: '#f3f4f6', color: '#374151' },
    required: { color: '#dc2626', marginLeft: 4 },
    methodBox: { background: '#f9fafb', padding: 12, borderRadius: 6, border: '1px solid #e5e7eb', marginBottom: 16 },
    formula: { fontSize: 11, color: '#6b7280', marginTop: 8, fontFamily: 'monospace', background: '#f3f4f6', padding: '4px 8px', borderRadius: 4, display: 'inline-block' },
    infoGrid: { display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: '16px', marginBottom: '24px', background: '#f8fafc', padding: '16px', borderRadius: '8px', border: '1px solid #e2e8f0' },
    infoItem: { fontSize: '13px' },
    infoLabel: { color: '#64748b', fontWeight: 500, display: 'block', marginBottom: '4px' },
    infoValue: { color: '#0f172a', fontWeight: 600, fontSize: '14px' },
  };

  // ============================================================================
  // RENDER
  // ============================================================================
  return (
    <div style={styles.container}>
      {/* HEADER */}
      <header style={styles.header}>
        <div>
          <div style={styles.title}>Лабораторная информационная система</div>
          <div style={styles.subtitle}>Ввод и регистрация результатов испытаний</div>
        </div>
        <span style={{
          ...styles.badge,
          ...(isCalculated && calculatedPreview !== undefined ? styles.badgeSuccess : styles.badgeNeutral),
        }}>
          {isCalculated && calculatedPreview !== undefined ? `Расчёт: ${formatNumber(calculatedPreview, currentMethod?.unit)}` : 'Готов к работе'}
        </span>
      </header>

      {/* MAIN */}
      <main style={styles.main}>
        <div style={styles.grid}>
          {/* LEFT: Параметры пробы */}
          <section style={styles.card}>
            <h3 style={styles.sectionTitle}>Параметры пробы</h3>
            <label style={styles.label}>
              <span style={styles.labelText}>Номер пробы *</span>
              <input style={styles.input} value={sampleNumber} onChange={e => setSampleNumber(e.target.value)} />
            </label>
            <label style={styles.label}>
              <span style={styles.labelText}>Место отбора *</span>
              <input style={styles.input} value={samplePlace} onChange={e => setSamplePlace(e.target.value)} placeholder="Например: Объект №5" />
            </label>
            <label style={styles.label}>
              <span style={styles.labelText}>Группа испытаний</span>
              <select style={styles.select} value={selectedGroupId} onChange={(e) => {
                const gid = e.target.value;
                setSelectedGroupId(gid);
                if (gid) {
                  const g = groups.find(grp => grp.id === gid);
                  if (g) { setSelectedMaterialId(g.material_id); loadStandards(g.material_id); }
                } else { setSelectedMaterialId(''); setStandards([]); setMethods([]); }
              }}>
                <option value="">-- Без группы --</option>
                {groups.map(g => <option key={g.id} value={g.id}>{g.name}</option>)}
              </select>
            </label>
            <button style={styles.btnSecondary} onClick={() => setIsGroupModalOpen(true)}>+ Создать группу</button>
            
            <div style={{ marginTop: 20, borderTop: '1px solid #e5e7eb', paddingTop: 16 }}>
              <label style={styles.label}>
                <span style={styles.labelText}>Лаборатория *</span>
                <input style={{...styles.input, borderColor: labName.trim() === '' ? '#ef4444' : '#d1d5db'}} value={labName} onChange={e => setLabName(e.target.value)} />
              </label>
              <label style={styles.label}>
                <span style={styles.labelText}>Оператор *</span>
                <input style={{...styles.input, borderColor: operator.trim() === '' ? '#ef4444' : '#d1d5db'}} value={operator} onChange={e => setOperator(e.target.value)} />
              </label>
            </div>
          </section>

          {/* RIGHT: Метод и ввод данных */}
          <section style={styles.card}>
            <h3 style={styles.sectionTitle}>Метод испытания</h3>
            <label style={styles.label}>
              <span style={styles.labelText}>Материал *</span>
              <select style={styles.select} value={selectedMaterialId} disabled={!!selectedGroupId} onChange={(e) => { setSelectedMaterialId(e.target.value); loadStandards(e.target.value); }}>
                <option value="">-- Выберите материал --</option>
                {materials.map(m => <option key={m.id} value={m.id}>{m.name}</option>)}
              </select>
            </label>
            <label style={styles.label}>
              <span style={styles.labelText}>Стандарт *</span>
              <select style={styles.select} value={selectedStandardId} disabled={!selectedMaterialId} onChange={(e) => { setSelectedStandardId(e.target.value); loadMethods(e.target.value); }}>
                <option value="">-- Выберите стандарт --</option>
                {standards.map(s => <option key={s.id} value={s.id}>{s.name}</option>)}
              </select>
            </label>
            <label style={styles.label}>
              <span style={styles.labelText}>Метод *</span>
              <select style={styles.select} value={selectedMethodId} disabled={!selectedStandardId} onChange={(e) => loadMethodDetails(e.target.value)}>
                <option value="">-- Выберите метод --</option>
                {methods.map(m => <option key={m.id} value={m.id}>{m.name} {m.unit ? `(${m.unit})` : ''}</option>)}
              </select>
            </label>

            {currentMethod ? (
              <div style={styles.methodBox}>
                <div style={{ fontWeight: 600, marginBottom: 12 }}>
                  {currentMethod.name} {currentMethod.unit && `(${currentMethod.unit})`}
                  {isCalculated && <span style={{ marginLeft: 8, fontSize: 11, color: '#6b7280' }}>(Расчётный)</span>}
                </div>
                {isCalculated ? (
                  <>
                    {currentMethod.inputs && currentMethod.inputs.length > 0 ? (
                      currentMethod.inputs.map((input: MethodInput) => (
                        <label key={input.param_key} style={styles.label}>
                          <span style={styles.labelText}>
                            {input.label} {input.unit && <span style={{ color: '#6b7280' }}>({input.unit})</span>}
                            {input.is_required && <span style={styles.required}>*</span>}
                          </span>
                          <input style={styles.input} type="number" step="any" value={rawInputs[input.param_key] || ''} onChange={(e) => setRawInputs({ ...rawInputs, [input.param_key]: e.target.value })} />
                        </label>
                      ))
                    ) : (<div style={{ color: '#6b7280', fontStyle: 'italic', padding: 10 }}>Нет параметров</div>)}
                    {currentMethod.formula_expr && <div style={styles.formula}>Формула: {currentMethod.formula_expr}</div>}
                    {calculatedPreview !== undefined && <div style={{ marginTop: 8, fontSize: 13, fontWeight: 600, color: '#166534' }}>Расчётное значение: {formatNumber(calculatedPreview, currentMethod?.unit)}</div>}
                  </>
                ) : (
                  <label style={styles.label}>
                    <span style={styles.labelText}>Результат {currentMethod.unit && <span style={{ color: '#6b7280' }}>({currentMethod.unit})</span>}</span>
                    <input style={styles.input} type="number" step="any" value={simpleValue} onChange={(e) => setSimpleValue(e.target.value)} />
                  </label>
                )}
              </div>
            ) : (
              <div style={{ color: '#6b7280', fontStyle: 'italic', padding: 20, textAlign: 'center', border: '1px dashed #d1d5db', borderRadius: 6 }}>Выберите метод для ввода результатов</div>
            )}
            <button style={styles.btnSecondary} onClick={() => { setRawInputs({}); setSimpleValue(''); setSelectedMethodId(''); }}>Сброс</button>
          </section>
        </div>

        {/* SAVE BUTTON */}
        <div style={{ marginTop: 24, display: 'flex', justifyContent: 'flex-end' }}>
          <button style={{ ...styles.btnPrimary, width: 'auto', minWidth: 200, opacity: canSave ? 1 : 0.5, cursor: canSave ? 'pointer' : 'not-allowed' }} disabled={!canSave || loading.save} onClick={handleSaveProtocol}>
            {loading.save ? 'Сохранение...' : 'Сохранить протокол'}
          </button>
        </div>

        {/* HISTORY */}
        <section style={{ ...styles.card, marginTop: 24 }}>
          <h3 style={styles.sectionTitle}>История протоколов</h3>
          {loading.protocols ? (
            <div style={{ textAlign: 'center', padding: 20, color: '#6b7280' }}>Загрузка...</div>
          ) : (
            <div style={{ overflowX: 'auto' }}>
              <table style={styles.table}>
                <thead>
                  <tr>
                    <th style={styles.th}>№ Протокола</th>
                    <th style={styles.th}>Дата</th>
                    <th style={styles.th}>Лаборатория</th>
                    <th style={styles.th}>Статус</th>
                    <th style={styles.th}>Действия</th>
                  </tr>
                </thead>
                <tbody>
                  {protocols.length === 0 ? (
                    <tr><td colSpan={5} style={{ textAlign: 'center', padding: 20, color: '#6b7280' }}>Нет протоколов</td></tr>
                  ) : protocols.map(p => (
                    <tr key={p.id}>
                      <td style={{...styles.td, fontWeight: 600, color: '#2563eb'}}>{p.protocol_number || p.id.substring(0, 8)}</td>
                      <td style={styles.td}>{formatDate(p.created_at)}</td>
                      <td style={styles.td}>{p.lab_name || '—'}</td>
                      <td style={styles.td}><span style={{...styles.badge, background: p.status === 'draft' ? '#fef3c7' : '#dcfce7', color: p.status === 'draft' ? '#92400e' : '#166534'}}>{p.status === 'draft' ? 'Черновик' : 'Завершен'}</span></td>
                      <td style={styles.td}>
                        <button style={styles.btnSmall} onClick={() => handleViewProtocol(p.id)}>Просмотр</button>
                        <button style={{...styles.btnSmall, marginLeft: 8}} onClick={() => handleDownloadProtocolPDF(p.id)}>PDF</button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>
      </main>

      {/* MODALS */}
      <Modal isOpen={isGroupModalOpen} onClose={() => setIsGroupModalOpen(false)} title="Новая группа">
        <label style={styles.label}>
          <span style={styles.labelText}>Название</span>
          <input style={styles.input} value={newGroup.name} onChange={e => setNewGroup({...newGroup, name: e.target.value})} />
        </label>
        <label style={styles.label}>
          <span style={styles.labelText}>Материал</span>
          <select style={styles.select} value={newGroup.materialId} onChange={e => setNewGroup({...newGroup, materialId: e.target.value})}>
            <option value="">-- Выберите --</option>
            {materials.map(m => <option key={m.id} value={m.id}>{m.name}</option>)}
          </select>
        </label>
        <div style={{ display: 'flex', gap: 12, marginTop: 20, justifyContent: 'flex-end' }}>
          <button style={{...styles.btnSecondary, width: 'auto', marginTop: 0}} onClick={() => setIsGroupModalOpen(false)}>Отмена</button>
          <button style={{...styles.btnPrimary, width: 'auto'}} onClick={handleCreateGroup}>Создать</button>
        </div>
      </Modal>

      {/* МОДАЛЬНОЕ ОКНО ПРОСМОТРА */}
      {viewingData && (
        <Modal isOpen={true} onClose={() => { setViewingData(null); setViewingResults([]); }} title="Просмотр протокола">
          
          <div style={styles.infoGrid}>
            <div style={styles.infoItem}>
              <span style={styles.infoLabel}>№ Протокола</span>
              <span style={{...styles.infoValue, fontSize: '16px', color: '#2563eb'}}>
                {viewingData.protocol.protocol_number || 'Не присвоен'}
              </span>
            </div>
            <div style={styles.infoItem}>
              <span style={styles.infoLabel}>Дата проведения</span>
              <span style={styles.infoValue}>{formatDate(viewingData.protocol.test_date || viewingData.protocol.created_at)}</span>
            </div>
            <div style={styles.infoItem}>
              <span style={styles.infoLabel}>Лаборатория</span>
              <span style={styles.infoValue}>{viewingData.protocol.lab_name || '—'}</span>
            </div>
            <div style={styles.infoItem}>
              <span style={styles.infoLabel}>Оператор</span>
              <span style={styles.infoValue}>{viewingData.protocol.operator_name || '—'}</span>
            </div>
            <div style={styles.infoItem}>
              <span style={styles.infoLabel}>Материал</span>
              <span style={styles.infoValue}>{viewingData.material.name || '—'}</span>
            </div>
            <div style={styles.infoItem}>
              <span style={styles.infoLabel}>Место отбора</span>
              <span style={styles.infoValue}>{viewingData.sample.collection_place || '—'}</span>
            </div>
            <div style={styles.infoItem}>
              <span style={styles.infoLabel}>Номер пробы</span>
              <span style={styles.infoValue}>{viewingData.sample.sample_number || '—'}</span>
            </div>
             <div style={styles.infoItem}>
              <span style={styles.infoLabel}>Примечание</span>
              <span style={{...styles.infoValue, fontWeight: 400, fontStyle: 'italic'}}>{viewingData.sample.note || '—'}</span>
            </div>
          </div>

          {viewingResults.length > 0 ? (
            <table style={styles.table}>
              <thead>
                <tr>
                  <th style={{...styles.th, width: '30%'}}>Метод</th>
                  <th style={{...styles.th, width: '15%'}}>Значение</th>
                  <th style={{...styles.th, width: '20%'}}>Норма</th>
                  <th style={{...styles.th, width: '15%'}}>Соответствие</th>
                  <th style={{...styles.th}}>Отклонение</th>
                </tr>
              </thead>
              <tbody>
                {viewingResults.map((r) => {
                  const isCompliant = r.is_compliant === true;
                  const isNonCompliant = r.is_compliant === false;
                  
                  return (
                    <tr key={r.id}>
                      <td style={{...styles.td, fontWeight: 500}}>
                        {r.methodName}
                        <div style={{fontSize: '10px', color: '#9ca3af', marginTop: '2px'}}>{r.method_id}</div>
                      </td>
                      <td style={styles.td}>{formatNumber(r.calculated_value)}</td>
                      <td style={{...styles.td, fontSize: '12px', color: '#6b7280'}}>{r.normString}</td>
                      <td style={styles.td}>
                        {r.is_compliant === undefined ? (
                          <span style={{color: '#6b7280'}}>—</span>
                        ) : isCompliant ? (
                          <span style={{...styles.badge, ...styles.badgeSuccess}}>Да</span>
                        ) : (
                          <span style={{...styles.badge, ...styles.badgeDanger}}>Нет</span>
                        )}
                      </td>
                      <td style={{...styles.td, color: isNonCompliant ? '#dc2626' : '#6b7280', fontSize: '12px', maxWidth: '150px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap'}}>
                        {isNonCompliant ? r.deviation_msg : '—'}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          ) : <div style={{padding: 20, textAlign: 'center', color: '#6b7280'}}>Нет данных о результатах</div>}

          <div style={{ display: 'flex', justifyContent: 'flex-end', marginTop: 24, gap: '12px' }}>
            <button style={{...styles.btnSecondary, width: 'auto', marginTop: 0}} onClick={() => { setViewingData(null); setViewingResults([]); }}>Закрыть</button>
            <button style={{...styles.btnPrimary, width: 'auto'}} onClick={() => handleDownloadProtocolPDF(viewingData.protocol.id)}>Скачать PDF</button>
          </div>
        </Modal>
      )}

      {toast && <Toast message={toast.msg} error={toast.error} onClose={() => setToast(null)} />}
    </div>
  );
}