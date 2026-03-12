// src/pages/LabDashboard.tsx
import { useState, useEffect, useMemo } from 'react';
import { models } from '../../wailsjs/go/models';
import {
  CreateGroup,
  CreateProtocolWithSample,
  GetGroupSummary,
  GetGroups,
  GetMaterials,
  GetMethodDetails,
  GetMethodsByStandardID,
  GetProtocolByID,
  GetProtocols,
  GetStandardsByMaterialID,
  GetStandardDimensions,
  SaveGroupPDFWithDialog,
  SaveProtocolPDFWithDialog,
} from '../../wailsjs/go/app/App';

// ============================================================================
// TYPE ALIASES
// ============================================================================
type Material = models.Material;
type Standard = models.Standard;
type TestMethod = models.TestMethod;
type MethodInput = models.MethodInputDTO;
type ContextDimension = models.ContextDimension;
type ExperimentGroup = models.ExperimentGroup;
type Protocol = models.Protocol;
type TestResult = models.TestResult;
type GroupSummary = models.GroupSummary;
type PaginatedMetadata = models.PaginatedMetadata;
type GroupListResponse = models.GroupListResponse;
type ProtocolListResponse = models.ProtocolListResponse;
type GetProtocolByIDRequest = models.GetProtocolByIDRequest;
type CreateProtocolRequest = models.CreateProtocolRequest;
type CreateSampleDTO = models.CreateSampleDTO;
type CreateResultDTO = models.CreateResultDTO;

// Расширяем TestMethod для полей, которые могут прийти с бэкенда
interface TestMethodWithInputs extends TestMethod {
  inputs?: MethodInput[];
  limits?: any[];
  min_value?: number;
  max_value?: number;
}

type CompliancePreview = {
  calculatedValue?: number;
  isCompliant: boolean | null;
  norm: string;
  deviation?: string;
};

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
    <button
      onClick={onClose}
      style={{ background: 'none', border: 'none', fontSize: 20, cursor: 'pointer', color: 'inherit', opacity: 0.6 }}
    >
      ×
    </button>
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
        top: 0,
        left: 0,
        right: 0,
        bottom: 0,
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
          maxWidth: 600,
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
  const [contextDimensions, setContextDimensions] = useState<ContextDimension[]>([]);

  // === STATE: Форма ===
  const [labName, setLabName] = useState('БГТУ им. В.Г. Шухова, Кафедра материаловедения');
  const [operator, setOperator] = useState('');
  const [sampleNumber, setSampleNumber] = useState('');
  const [samplePlace, setSamplePlace] = useState('');
  const [sampleNote, setSampleNote] = useState('');
  const [sampleContext, setSampleContext] = useState<Record<string, string>>({});
  const [selectedGroupId, setSelectedGroupId] = useState<string>('');
  const [selectedMaterialId, setSelectedMaterialId] = useState<string>('');
  const [selectedStandardId, setSelectedStandardId] = useState<string>('');
  const [selectedMethodId, setSelectedMethodId] = useState<string>('');

  // Ввод результатов
  const [simpleValue, setSimpleValue] = useState<string>('');
  const [rawInputs, setRawInputs] = useState<Record<string, string>>({});

  // === STATE: Списки ===
  const [groups, setGroups] = useState<ExperimentGroup[]>([]);
  const [groupsMeta, setGroupsMeta] = useState<PaginatedMetadata | null>(null);
  const [groupsPage, setGroupsPage] = useState(1);
  const [protocols, setProtocols] = useState<Protocol[]>([]);
  const [protocolsMeta, setProtocolsMeta] = useState<PaginatedMetadata | null>(null);
  const [protocolsPage, setProtocolsPage] = useState(1);

  // === STATE: UI ===
  const [loading, setLoading] = useState<Record<string, boolean>>({
    materials: true,
    groups: true,
    protocols: true,
    standards: false,
    methods: false,
    save: false,
  });
  const [isGroupModalOpen, setIsGroupModalOpen] = useState(false);
  const [newGroup, setNewGroup] = useState({ name: '', materialId: '', project: '', location: '' });
  const [viewingProtocol, setViewingProtocol] = useState<GetProtocolByIDRequest | null>(null);
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
      setGroupsMeta(response.meta || null);
      setGroupsPage(page);
    }
    setLoading((p) => ({ ...p, groups: false }));
  };

  const loadProtocolsList = async (page: number) => {
    setLoading((p) => ({ ...p, protocols: true }));
    const offset = (page - 1) * 10;
    const response = await safeCall(() => GetProtocols(10, offset) as Promise<ProtocolListResponse>, 'Ошибка загрузки протоколов');
    if (response) {
      setProtocols(response.items || []);
      setProtocolsMeta(response.meta || null);
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

  // КЛЮЧЕВАЯ ФУНКЦИЯ: Загрузка деталей метода с inputs
  const loadMethodDetails = async (methodId: string) => {
    if (!methodId) return;
    setSelectedMethodId(methodId);
    setRawInputs({});
    setSimpleValue('');
    
    // GetMethodDetails возвращает TestMethodFull
    const details = await safeCall(() => GetMethodDetails(methodId), 'Ошибка загрузки деталей метода');
    
    if (details) {
      // ✅ Доступ через .method для свойств TestMethod
      const extended: TestMethodWithInputs = {
        id: details.method.id,                    // ← через .method
        standard_id: details.method.standard_id,  // ← через .method
        name: details.method.name,                // ← через .method
        unit: details.method.unit,                // ← через .method
        result_type: details.method.result_type,
        is_mandatory: details.method.is_mandatory,
        code: details.method.code,
        description: details.method.description,
        formula_expr: details.method.formula_expr,
        // Дополнительные поля из TestMethodFull
        inputs: details.inputs ?? [],             // ← прямо, т.к. это поле TestMethodFull
        limits: details.limits,
        min_value: (details as any).min_value,
        max_value: (details as any).max_value,
      };
      
      setMethods((prev) => prev.map((m) => (m.id === methodId ? { ...m, ...extended } : m)));
    }
  };

  // === COMPUTED ===
  const currentMethod = useMemo(() => methods.find((m) => m.id === selectedMethodId), [methods, selectedMethodId]);

  // Определяем тип метода: расчётный (с inputs) или простой
  const isCalculated = useMemo(() => {
    return !!(currentMethod?.inputs && currentMethod.inputs.length > 0);
  }, [currentMethod]);

  // Проверка заполнения всех обязательных полей
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

  // Расчёт значения по формуле (превью)
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

  const canSave = !!(sampleNumber && samplePlace && selectedMaterialId && selectedMethodId && allInputsFilled);

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
      loadGroupsList(groupsPage);
    }
  };

  // ГЛАВНАЯ ФУНКЦИЯ: СОХРАНЕНИЕ ПРОТОКОЛА
  const handleSaveProtocol = async () => {
    if (!canSave) { showToast('Заполните все обязательные поля', true); return; }
    const method = methods.find((m) => m.id === selectedMethodId);
    if (!method) { showToast('Метод не найден', true); return; }

    let resultsInput: CreateResultDTO[] = [];
    if (isCalculated) {
      // Расчётный метод: отправляем raw_inputs
      for (const p of currentMethod?.inputs ?? []) {
        const valStr = rawInputs[p.param_key];
        if (p.is_required && (!valStr || valStr.trim() === '')) {
          showToast(`Заполните параметр: ${p.label}`, true);
          return;
        }
        if (p.input_type === 'number' && valStr && isNaN(parseFloat(valStr))) {
          showToast(`Некорректное число: ${p.label}`, true);
          return;
        }
      }
      resultsInput = [{
        method_id: method.id,
        raw_inputs: { ...rawInputs },
        note: undefined,
      } as CreateResultDTO];
    } else {
      // Простой метод: отправляем готовое значение
      const val = parseFloat(simpleValue);
      if (isNaN(val)) { showToast('Введите корректное число', true); return; }
      resultsInput = [{
        method_id: method.id,
        raw_inputs: { value: val.toString() },
        note: undefined,
      } as CreateResultDTO];
    }

    const sampleData = models.CreateSampleDTO.createFrom({
      sample_number: sampleNumber,
      material_id: selectedMaterialId,
      collection_date: undefined,
      context_params: undefined,
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
      // Сброс формы
      setRawInputs({});
      setSimpleValue('');
      setSampleContext({});
      setSelectedMethodId('');
      setSampleNumber('');
      setSamplePlace('');
      setSampleNote('');
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

  const handleViewProtocol = async (id: string) => {
    setLoading((p) => ({ ...p, save: true }));
    const protocol = await safeCall(() => GetProtocolByID(id), 'Ошибка загрузки');
    setLoading((p) => ({ ...p, save: false }));
    if (protocol) setViewingProtocol(protocol);
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
    badge: { display: 'inline-block', padding: '2px 8px', borderRadius: 999, fontSize: 11, fontWeight: 600, background: '#dbeafe', color: '#1e40af' },
    badgeSuccess: { background: '#dcfce7', color: '#166534' },
    badgeDanger: { background: '#fee2e2', color: '#991b1b' },
    required: { color: '#dc2626', marginLeft: 4 },
    methodBox: { background: '#f9fafb', padding: 12, borderRadius: 6, border: '1px solid #e5e7eb', marginBottom: 16 },
    formula: { fontSize: 11, color: '#6b7280', marginTop: 8, fontFamily: 'monospace', background: '#f3f4f6', padding: '4px 8px', borderRadius: 4, display: 'inline-block' },
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
          ...(isCalculated && calculatedPreview !== undefined ? styles.badgeSuccess : {}),
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
              <input style={styles.input} value={samplePlace} onChange={e => setSamplePlace(e.target.value)} />
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
                <span style={styles.labelText}>Лаборатория</span>
                <input style={styles.input} value={labName} onChange={e => setLabName(e.target.value)} />
              </label>
              <label style={styles.label}>
                <span style={styles.labelText}>Оператор</span>
                <input style={styles.input} value={operator} onChange={e => setOperator(e.target.value)} />
              </label>
            </div>
          </section>

          {/* RIGHT: Метод и ввод данных */}
          <section style={styles.card}>
            <h3 style={styles.sectionTitle}>Метод испытания</h3>
            <label style={styles.label}>
              <span style={styles.labelText}>Материал *</span>
              <select style={styles.select} value={selectedMaterialId} disabled={!!selectedGroupId}
                onChange={(e) => { setSelectedMaterialId(e.target.value); loadStandards(e.target.value); }}>
                <option value="">-- Выберите материал --</option>
                {materials.map(m => <option key={m.id} value={m.id}>{m.name}</option>)}
              </select>
            </label>
            <label style={styles.label}>
              <span style={styles.labelText}>Стандарт *</span>
              <select style={styles.select} value={selectedStandardId} disabled={!selectedMaterialId}
                onChange={(e) => { setSelectedStandardId(e.target.value); loadMethods(e.target.value); }}>
                <option value="">-- Выберите стандарт --</option>
                {standards.map(s => <option key={s.id} value={s.id}>{s.name}</option>)}
              </select>
            </label>
            <label style={styles.label}>
              <span style={styles.labelText}>Метод *</span>
              <select style={styles.select} value={selectedMethodId} disabled={!selectedStandardId}
                onChange={(e) => loadMethodDetails(e.target.value)}>
                <option value="">-- Выберите метод --</option>
                {methods.map(m => <option key={m.id} value={m.id}>{m.name} {m.unit ? `(${m.unit})` : ''}</option>)}
              </select>
            </label>

            {/* ПОЛЯ ВВОДА ДЛЯ МЕТОДА */}
            {currentMethod ? (
              <div style={styles.methodBox}>
                <div style={{ fontWeight: 600, marginBottom: 12 }}>
                  {currentMethod.name} {currentMethod.unit && `(${currentMethod.unit})`}
                  {isCalculated && <span style={{ marginLeft: 8, fontSize: 11, color: '#6b7280' }}>(Расчётный)</span>}
                </div>
                {isCalculated ? (
                  // РАСЧЁТНЫЙ МЕТОД: показываем все inputs
                  <>
                    {currentMethod.inputs && currentMethod.inputs.length > 0 ? (
                      currentMethod.inputs.map((input: MethodInput) => (
                        <label key={input.param_key} style={styles.label}>
                          <span style={styles.labelText}>
                            {input.label} {input.unit && <span style={{ color: '#6b7280' }}>({input.unit})</span>}
                            {input.is_required && <span style={styles.required}>*</span>}
                          </span>
                          <input
                            style={styles.input}
                            type="number"
                            step="any"
                            value={rawInputs[input.param_key] || ''}
                            onChange={(e) => setRawInputs({ ...rawInputs, [input.param_key]: e.target.value })}
                            placeholder="Введите значение"
                          />
                        </label>
                      ))
                    ) : (
                      <div style={{ color: '#6b7280', fontStyle: 'italic', padding: 10 }}>
                        Нет параметров для ввода
                      </div>
                    )}
                    {currentMethod.formula_expr && (
                      <div style={styles.formula}>Формула: {currentMethod.formula_expr}</div>
                    )}
                    {calculatedPreview !== undefined && (
                      <div style={{ marginTop: 8, fontSize: 13, fontWeight: 600, color: '#166534' }}>
                        Расчётное значение: {formatNumber(calculatedPreview, currentMethod?.unit)}
                      </div>
                    )}
                  </>
                ) : (
                  // ПРОСТОЙ МЕТОД: одно поле
                  <label style={styles.label}>
                    <span style={styles.labelText}>
                      Результат {currentMethod.unit && <span style={{ color: '#6b7280' }}>({currentMethod.unit})</span>}
                    </span>
                    <input
                      style={styles.input}
                      type="number"
                      step="any"
                      value={simpleValue}
                      onChange={(e) => setSimpleValue(e.target.value)}
                    />
                  </label>
                )}
              </div>
            ) : (
              <div style={{ color: '#6b7280', fontStyle: 'italic', padding: 20, textAlign: 'center', border: '1px dashed #d1d5db', borderRadius: 6 }}>
                Выберите метод для ввода результатов
              </div>
            )}
            <button style={styles.btnSecondary} onClick={() => {
              setRawInputs({});
              setSimpleValue('');
              setSelectedMethodId('');
            }}>
              Сброс
            </button>
          </section>
        </div>

        {/* SAVE BUTTON */}
        <div style={{ marginTop: 24, display: 'flex', justifyContent: 'flex-end' }}>
          <button
            style={{
              ...styles.btnPrimary,
              width: 'auto',
              minWidth: 200,
              opacity: canSave ? 1 : 0.5,
              cursor: canSave ? 'pointer' : 'not-allowed'
            }}
            disabled={!canSave || loading.save}
            onClick={handleSaveProtocol}
          >
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
                    <th style={styles.th}>Дата</th>
                    <th style={styles.th}>Лаборатория</th>
                    <th style={styles.th}>Статус</th>
                    <th style={styles.th}>Действия</th>
                  </tr>
                </thead>
                <tbody>
                  {protocols.length === 0 ? (
                    <tr><td colSpan={4} style={{ textAlign: 'center', padding: 20, color: '#6b7280' }}>Нет протоколов</td></tr>
                  ) : protocols.map(p => (
                    <tr key={p.id}>
                      <td style={styles.td}>{formatDate(p.created_at)}</td>
                      <td style={styles.td}>{p.lab_name || '—'}</td>
                      <td style={styles.td}><span style={styles.badge}>{p.status}</span></td>
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

      {viewingProtocol && (
        <Modal isOpen={true} onClose={() => setViewingProtocol(null)} title="Протокол">
          <div style={{ marginBottom: 16, fontSize: 13 }}>
            <div><strong>ID:</strong> {viewingProtocol.Protocol.id}</div>
            <div><strong>Дата:</strong> {formatDate(viewingProtocol.Protocol.created_at)}</div>
            <div><strong>Статус:</strong> {viewingProtocol.Protocol.status}</div>
          </div>
          {viewingProtocol.Results?.length ? (
            <table style={styles.table}>
              <thead>
                <tr><th style={styles.th}>Метод</th><th style={styles.th}>Значение</th><th style={styles.th}>Статус</th></tr>
              </thead>
              <tbody>
                {viewingProtocol.Results.map(r => (
                  <tr key={r.id}>
                    <td style={styles.td}>{r.method_id}</td>
                    <td style={styles.td}>{formatNumber(r.calculated_value)}</td>
                    <td style={styles.td}>
                      {r.is_compliant === true ? <span style={{color: '#166534'}}>Да</span> :
                       r.is_compliant === false ? <span style={{color: '#991b1b'}}>Нет</span> : '—'}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          ) : <div style={{padding: 20, textAlign: 'center', color: '#6b7280'}}>Нет данных</div>}
          <div style={{ display: 'flex', justifyContent: 'flex-end', marginTop: 20 }}>
            <button style={styles.btnPrimary} onClick={() => handleDownloadProtocolPDF(viewingProtocol.Protocol.id)}>Скачать PDF</button>
          </div>
        </Modal>
      )}

      {toast && <Toast message={toast.msg} error={toast.error} onClose={() => setToast(null)} />}
    </div>
  );
}