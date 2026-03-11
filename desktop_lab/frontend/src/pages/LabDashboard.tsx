// src/pages/LabDashboard.tsx
import { useState, useEffect, useMemo, useCallback } from 'react';
import { models } from '../../wailsjs/go/models';
import {
  CreateGroup,
  CreateProtocolWithSample,
  GenerateProtocolPDF,
  GenerateGroupSummaryPDF,
  GetGroupByID,
  GetGroupSummary,
  GetGroups,
  GetMaterialByID,
  GetMaterials,
  GetMethodDetails,
  GetMethodsByStandardID,
  GetProtocolByID,
  GetProtocols,
  GetStandardsByMaterialID,
  SaveGroupPDFWithDialog,
  SaveProtocolPDFWithDialog,
} from '../../wailsjs/go/app/App';

// Type aliases для удобства
type Material = models.Material;
type Standard = models.Standard;
type TestMethod = models.TestMethod;
type MethodInput = models.MethodInput;
type ContextDimension = models.ContextDimension;
type NormativeLimit = models.NormativeLimit;
type LimitCondition = models.LimitCondition;
type ExperimentGroup = models.ExperimentGroup;
type Sample = models.Sample;
type Protocol = models.Protocol;
type ProtocolListItem = models.ProtocolListItem;
type TestResult = models.TestResult;
type CreateProtocolRequest = models.CreateProtocolRequest;
type CreateSampleDTO = models.CreateSampleDTO;
type CreateResultDTO = models.CreateResultDTO;
type GroupSummary = models.GroupSummary;
type MethodResultSummary = models.MethodResultSummary;
type MethodTrial = models.MethodTrial;
type PaginatedMetadata = models.PaginatedMetadata;
type GroupListResponse = models.GroupListResponse;
type ProtocolListResponse = models.ProtocolListResponse;

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

const formatNumber = (value: number | undefined, unit: string = ''): string => {
  if (value === undefined || value === null) return '—';
  return `${value.toFixed(2)}${unit ? ` ${unit}` : ''}`;
};

// ============================================================================
// КОМПОНЕНТЫ ИНТЕРФЕЙСА
// ============================================================================

// Тост уведомления
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
      borderRadius: 10,
      boxShadow: '0 10px 25px -5px rgba(0, 0, 0, 0.1)',
      zIndex: 1000,
      display: 'flex',
      alignItems: 'center',
      gap: 10,
      fontSize: 14,
      color: error ? '#991b1b' : '#166534',
      minWidth: 280,
      maxWidth: 400,
    }}
  >
    <span>{error ? '❌' : '✅'}</span>
    <span style={{ flex: 1 }}>{message}</span>
    <button
      onClick={onClose}
      style={{
        background: 'none',
        border: 'none',
        fontSize: 18,
        cursor: 'pointer',
        color: 'inherit',
        opacity: 0.7,
        padding: '0 4px',
      }}
    >
      ×
    </button>
  </div>
);

// Модалка
const Modal: React.FC<{
  isOpen: boolean;
  onClose: () => void;
  title: string;
  children: React.ReactNode;
  size?: 'sm' | 'md' | 'lg' | 'xl';
}> = ({ isOpen, onClose, title, children, size = 'md' }) => {
  if (!isOpen) return null;

  const sizes: Record<string, string> = {
    sm: 'max-w-md',
    md: 'max-w-lg',
    lg: 'max-w-2xl',
    xl: 'max-w-4xl',
  };

  return (
    <div
      style={{
        position: 'fixed',
        top: 0,
        left: 0,
        right: 0,
        bottom: 0,
        background: 'rgba(15, 23, 42, 0.6)',
        zIndex: 100,
        display: 'grid',
        placeItems: 'center',
        backdropFilter: 'blur(4px)',
      }}
      onClick={onClose}
    >
      <div
        style={{
          background: '#fff',
          padding: 24,
          borderRadius: 16,
          width: '95%',
          maxWidth: sizes[size],
          boxShadow: '0 25px 50px -12px rgba(0, 0, 0, 0.25)',
          maxHeight: '90vh',
          overflowY: 'auto',
        }}
        onClick={(e) => e.stopPropagation()}
      >
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 20 }}>
          <h3 style={{ margin: 0, fontSize: 18, fontWeight: 600, color: '#1e293b' }}>{title}</h3>
          <button
            onClick={onClose}
            style={{
              background: 'none',
              border: 'none',
              fontSize: 24,
              cursor: 'pointer',
              color: '#64748b',
              padding: '4px 8px',
              borderRadius: 6,
              lineHeight: 1,
            }}
          >
            ×
          </button>
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
  const [methods, setMethods] = useState<TestMethod[]>([]);
  const [contextDimensions, setContextDimensions] = useState<ContextDimension[]>([]);

  // === STATE: Формы ===
  const [labName, setLabName] = useState('БГТУ им. В.Г. Шухова, Кафедра материаловедения');
  const [operator, setOperator] = useState('');
  const [project, setProject] = useState('');
  
  const [sampleNumber, setSampleNumber] = useState('');
  const [samplePlace, setSamplePlace] = useState('');
  const [sampleNote, setSampleNote] = useState('');
  const [sampleContext, setSampleContext] = useState<Record<string, string>>({});
  
  const [selectedGroupId, setSelectedGroupId] = useState<string>('');
  const [selectedMaterialId, setSelectedMaterialId] = useState<string>('');
  const [selectedStandardId, setSelectedStandardId] = useState<string>('');
  const [selectedMethodId, setSelectedMethodId] = useState<string>('');
  const [rawInputs, setRawInputs] = useState<Record<string, string>>({});

  // === STATE: Списки ===
  const [groups, setGroups] = useState<ExperimentGroup[]>([]);
  const [groupsMeta, setGroupsMeta] = useState<PaginatedMetadata | null>(null);
  const [groupsPage, setGroupsPage] = useState(1);
  
  const [protocols, setProtocols] = useState<ProtocolListItem[]>([]);
  const [protocolsMeta, setProtocolsMeta] = useState<PaginatedMetadata | null>(null);
  const [protocolsPage, setProtocolsPage] = useState(1);

  // === STATE: Загрузка ===
  const [loading, setLoading] = useState<Record<string, boolean>>({
    materials: true,
    groups: true,
    protocols: true,
    standards: false,
    methods: false,
    save: false,
  });

  // === STATE: Модалки и уведомления ===
  const [isGroupModalOpen, setIsGroupModalOpen] = useState(false);
  const [newGroup, setNewGroup] = useState({ name: '', materialId: '', project: '', location: '' });
  
  const [viewingProtocol, setViewingProtocol] = useState<Protocol | null>(null);
  const [viewingSummary, setViewingSummary] = useState<GroupSummary | null>(null);
  
  const [toast, setToast] = useState<{ msg: string; error: boolean } | null>(null);

  // === ИНИЦИАЛИЗАЦИЯ ===
  useEffect(() => {
    loadMaterials();
    loadGroupsList(1);
    loadProtocolsList(1);
  }, []);

  // === ЗАГРУЗКА ДАННЫХ ===
  const loadMaterials = async () => {
    setLoading((p) => ({ ...p, materials: true }));
    const list = await safeCall(() => GetMaterials(), 'Ошибка загрузки материалов');
    if (list) setMaterials(list);
    setLoading((p) => ({ ...p, materials: false }));
  };

  const loadGroupsList = async (page: number) => {
    setLoading((p) => ({ ...p, groups: true }));
    const offset = (page - 1) * 10;
    const response = await safeCall(
      () => GetGroups(10, offset) as Promise<GroupListResponse>,
      'Ошибка загрузки групп'
    );
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
    const response = await safeCall(
      () => GetProtocols(10, offset) as Promise<ProtocolListResponse>,
      'Ошибка загрузки протоколов'
    );
    if (response) {
      setProtocols(response.items || []);
      setProtocolsMeta(response.meta || null);
      setProtocolsPage(page);
    }
    setLoading((p) => ({ ...p, protocols: false }));
  };

  const loadStandards = async (materialId: string) => {
    console.log("🔍 Попытка загрузить стандарты для материала:", materialId); // <--- ДОБАВИТЬ
    
    if (!materialId) {
      setStandards([]);
      setContextDimensions([]);
      return;
    }
    setLoading((p) => ({ ...p, standards: true }));
    
    try {
        // Попробуй вызвать без safeCall временно, чтобы увидеть полную ошибку в консоли
        const list = await GetStandardsByMaterialID(materialId); 
        console.log("✅ Получены стандарты:", list); // <--- ДОБАВИТЬ
        
        if (list) {
          setStandards(list);
          const allDims: ContextDimension[] = [];
          list.forEach((std) => {
            if (std.dimensions) allDims.push(...std.dimensions);
          });
          setContextDimensions(allDims);
          setMethods([]);
          setSelectedStandardId('');
          setSelectedMethodId('');
          setSampleContext({});
        }
    } catch (e) {
        console.error("❌ Критическая ошибка при загрузке стандартов:", e); // <--- ДОБАВИТЬ
        alert("Ошибка загрузки стандартов: " + e); // Всплывающее окно для явной видимости
    } finally {
        setLoading((p) => ({ ...p, standards: false }));
    }
  };

  const loadMethods = async (standardId: string) => {
    if (!standardId) {
      setMethods([]);
      return;
    }
    setLoading((p) => ({ ...p, methods: true }));
    const list = await safeCall(
      () => GetMethodsByStandardID(standardId) as Promise<TestMethod[]>,
      'Ошибка загрузки методов'
    );
    if (list) {
      setMethods(list);
      setSelectedMethodId('');
      setRawInputs({});
    }
    setLoading((p) => ({ ...p, methods: false }));
  };

  // === ВЫЧИСЛЯЕМЫЕ ЗНАЧЕНИЯ ===
  const currentMethod = useMemo(
    () => methods.find((m) => m.id === selectedMethodId),
    [methods, selectedMethodId]
  );

  const requiredInputs = useMemo(
    () => (currentMethod?.inputs || []).filter((inp) => inp.is_required),
    [currentMethod]
  );

  const allInputsFilled = useMemo(() => {
    return requiredInputs.every((inp) => {
      const val = rawInputs[inp.param_key];
      return val && val.trim() !== '';
    });
  }, [requiredInputs, rawInputs]);

  const previewCompliance = useMemo(() => {
    if (!currentMethod || !allInputsFilled) return null;

    // Парсим входные данные
    const inputs: Record<string, number> = {};
    for (const inp of currentMethod.inputs || []) {
      if (inp.input_type === 'number' && rawInputs[inp.param_key]) {
        const val = parseFloat(rawInputs[inp.param_key]);
        if (!isNaN(val)) inputs[inp.param_key] = val;
      }
    }

    // Если есть формула — считаем (упрощённо)
    let calculatedValue: number | undefined;
    if (currentMethod.formula_expr) {
      try {
        // Простая эвристика для демонстрации
        if (currentMethod.formula_expr.includes('/') && inputs['F'] && inputs['A']) {
          calculatedValue = inputs['F'] / inputs['A'];
        } else if (currentMethod.formula_expr.includes('*') && inputs['F'] && inputs['A']) {
          calculatedValue = inputs['F'] * inputs['A'];
        } else if (inputs['value']) {
          calculatedValue = inputs['value'];
        }
        if (calculatedValue !== undefined) {
          calculatedValue = Math.round(calculatedValue * 100) / 100;
        }
      } catch {
        // Игнорируем ошибки расчёта в превью
      }
    }

    // Ищем подходящий лимит по контексту
    const limit = findApplicableLimit(currentMethod, sampleContext);
    
    if (!limit || calculatedValue === undefined) {
      return { calculatedValue, isCompliant: null, norm: '—', deviation: undefined };
    }

    // Проверяем соответствие
    let isCompliant = true;
    let norm = '—';
    let deviation: string | undefined;

    if (limit.limit_type === 'range' && limit.min_value != null && limit.max_value != null) {
      norm = `${limit.min_value} – ${limit.max_value}`;
      if (calculatedValue < limit.min_value || calculatedValue > limit.max_value) {
        isCompliant = false;
        deviation = calculatedValue < limit.min_value
          ? `Ниже нормы на ${(limit.min_value - calculatedValue).toFixed(2)}`
          : `Выше нормы на ${(calculatedValue - limit.max_value).toFixed(2)}`;
      }
    } else if (limit.limit_type === 'min' && limit.min_value != null) {
      norm = `≥ ${limit.min_value}`;
      if (calculatedValue < limit.min_value) {
        isCompliant = false;
        deviation = `Ниже нормы на ${(limit.min_value - calculatedValue).toFixed(2)}`;
      }
    } else if (limit.limit_type === 'max' && limit.max_value != null) {
      norm = `≤ ${limit.max_value}`;
      if (calculatedValue > limit.max_value) {
        isCompliant = false;
        deviation = `Выше нормы на ${(calculatedValue - limit.max_value).toFixed(2)}`;
      }
    }

    return { calculatedValue, isCompliant, norm, deviation };
  }, [currentMethod, rawInputs, sampleContext, allInputsFilled]);

  const canSave = useMemo(() => {
    return sampleNumber && samplePlace && selectedMaterialId && selectedMethodId && allInputsFilled;
  }, [sampleNumber, samplePlace, selectedMaterialId, selectedMethodId, allInputsFilled]);

  // === ЛОГИКА ВАЛИДАЦИИ ===
  const findApplicableLimit = (method: TestMethod, context: Record<string, string>): NormativeLimit | null => {
    if (!method.limits) return null;
    
    for (const limit of method.limits) {
      if (!limit.conditions?.length) return limit; // Если нет условий — берём первый
      
      const matches = limit.conditions.every((cond) => {
        const actual = context[cond.dimension_key];
        return actual && actual === cond.expected_value;
      });
      
      if (matches) return limit;
    }
    return null;
  };

  // === ОБРАБОТЧИКИ ===
  const showToast = (msg: string, error = false) => {
    setToast({ msg, error });
    setTimeout(() => setToast(null), error ? 5000 : 3000);
  };

  const handleCreateGroup = async () => {
    if (!newGroup.name || !newGroup.materialId) {
      showToast('Заполните название и материал группы', true);
      return;
    }
    setLoading((p) => ({ ...p, save: true }));
    const result = await safeCall(
      () => CreateGroup(newGroup.name, newGroup.project, newGroup.location || '', newGroup.materialId),
      'Ошибка создания группы'
    );
    setLoading((p) => ({ ...p, save: false }));
    
    if (result) {
      showToast('Группа создана!');
      setIsGroupModalOpen(false);
      setNewGroup({ name: '', materialId: '', project: '', location: '' });
      loadGroupsList(groupsPage);
    }
  };

  // === SAVE HANDLER ===
  const handleSaveProtocol = async () => {
    // 1. Базовая валидация
    if (!sampleNumber || !samplePlace) {
      showToast("Заполните номер пробы и место отбора", true);
      return;
    }
    if (!selectedMaterialId || !selectedMethodId) {
      showToast("Выберите материал и метод", true);
      return;
    }

    const method = methods.find((m: TestMethod) => m.id === selectedMethodId);
    if (!method) {
      showToast("Метод не найден", true);
      return;
    }

    // 2. Валидация обязательных полей метода
    const allInputs = method.inputs || [];
    for (const p of allInputs) {
      const valStr = rawInputs[p.param_key];
      if (p.is_required && (!valStr || valStr.trim() === '')) {
        showToast(`Заполните параметр: ${p.label}`, true);
        return;
      }
      if (p.input_type === 'number' && valStr && valStr.trim() !== '' && isNaN(parseFloat(valStr))) {
        showToast(`Некорректное число для "${p.label}"`, true);
        return;
      }
    }

    // 3. Создаём объекты через createFrom (автогенерированные Wails модели)
    const resultsInput = [
      models.CreateResultDTO.createFrom({
        method_id: method.id,
        raw_inputs: { ...rawInputs },
        note: undefined
      })
    ];

    const fullContext = {
      ...sampleContext,
      location: samplePlace
    };

    const sampleData = models.CreateSampleDTO.createFrom({
      sample_number: sampleNumber,
      material_id: selectedMaterialId,
      collection_date: undefined,
      context_params: fullContext,
      note: sampleNote || undefined
    });

    const reqData = models.CreateProtocolRequest.createFrom({
      group_id: selectedGroupId || "",
      sample: sampleData,
      lab_name: labName,
      operator_name: operator,
      results: resultsInput
    });

    // 4. Отправка на бэкенд
    try {
       setLoading({ save: true });
      await CreateProtocolWithSample(reqData);
      showToast("Протокол сохранён!");
      
      // 5. Сброс формы
      setRawInputs({});
      setSampleContext({});
      setSelectedMethodId("");
      setSampleNumber("");
      setSamplePlace("");
      setSampleNote("");
      
      // 6. Обновление списка
      loadProtocolsList(1);
    } catch (e: any) {
      console.error("Ошибка сохранения:", e);
      const errorMsg = e instanceof Error ? e.message : String(e);
      showToast("Ошибка: " + errorMsg, true);
    } finally {
      setLoading({ save: false });
    }
  };

  const handleDownloadProtocolPDF = async (id: string) => {
    setLoading((p) => ({ ...p, save: true }));
    const path = await safeCall(
      () => SaveProtocolPDFWithDialog(id),
      'Ошибка генерации PDF'
    );
    setLoading((p) => ({ ...p, save: false }));
    
    if (path) {
      const fileName = path.split('/').pop() || path.split('\\').pop() || path;
      showToast(`Файл сохранён: ${fileName}`);
    }
  };

  const handleDownloadGroupPDF = async (id: string) => {
    setLoading((p) => ({ ...p, save: true }));
    const path = await safeCall(
      () => SaveGroupPDFWithDialog(id),
      'Ошибка генерации отчёта'
    );
    setLoading((p) => ({ ...p, save: false }));
    
    if (path) {
      const fileName = path.split('/').pop() || path.split('\\').pop() || path;
      showToast(`Файл сохранён: ${fileName}`);
    }
  };

  const handleViewGroupSummary = async (groupId: string) => {
    setLoading((p) => ({ ...p, save: true }));
    const summary = await safeCall(
      () => GetGroupSummary(groupId),
      'Ошибка загрузки сводки'
    );
    setLoading((p) => ({ ...p, save: false }));
    
    if (summary) setViewingSummary(summary);
  };

  const handleViewProtocol = async (protocolId: string) => {
    setLoading((p) => ({ ...p, save: true }));
    const protocol = await safeCall(
      () => GetProtocolByID(protocolId),
      'Ошибка загрузки протокола'
    );
    setLoading((p) => ({ ...p, save: false }));
    
    if (protocol) setViewingProtocol(protocol);
  };

  // === СТИЛИ ===
  const styles = {
    container: {
      display: 'flex',
      flexDirection: 'column' as const,
      height: '100vh',
      background: '#f8fafc',
      color: '#0f172a',
      fontFamily: 'system-ui, -apple-system, sans-serif',
    },
    header: {
      position: 'sticky' as const,
      top: 0,
      zIndex: 10,
      background: 'rgba(255,255,255,0.95)',
      backdropFilter: 'blur(8px)',
      borderBottom: '1px solid #e2e8f0',
    },
    wrap: {
      maxWidth: 1400,
      margin: '0 auto',
      padding: '12px 24px',
      width: '100%',
      boxSizing: 'border-box' as const,
    },
    brand: {
      display: 'flex',
      alignItems: 'center',
      gap: 12,
    },
    logo: {
      width: 40,
      height: 40,
      borderRadius: 10,
      background: 'linear-gradient(135deg, #0f172a, #1e293b)',
      color: '#fff',
      display: 'grid',
      placeItems: 'center',
      fontWeight: 700,
      fontSize: 14,
    },
    muted: { color: '#64748b', fontSize: 12 },
    row: {
      display: 'flex',
      gap: 12,
      alignItems: 'center',
      justifyContent: 'space-between',
    },
    badge: {
      display: 'inline-flex',
      alignItems: 'center',
      fontSize: 12,
      padding: '5px 12px',
      borderRadius: 999,
      background: '#e2e8f0',
      color: '#0f172a',
      fontWeight: 500,
    },
    bGreen: {
      background: '#dcfce7',
      color: '#166534',
      padding: '3px 8px',
      borderRadius: 4,
      fontSize: 11,
      fontWeight: 500,
    },
    bRed: {
      background: '#fee2e2',
      color: '#991b1b',
      padding: '3px 8px',
      borderRadius: 4,
      fontSize: 11,
      fontWeight: 500,
    },
    gridMain: {
      display: 'grid',
      gap: 20,
      gridTemplateColumns: 'repeat(auto-fit, minmax(320px, 1fr))',
    },
    gridHistory: {
      display: 'grid',
      gap: 20,
      gridTemplateColumns: 'repeat(auto-fit, minmax(450px, 1fr))',
      marginTop: 20,
    },
    card: {
      background: '#fff',
      border: '1px solid #e2e8f0',
      borderRadius: 12,
      padding: 20,
      boxShadow: '0 1px 3px rgba(0,0,0,0.04)',
    },
    title: {
      fontWeight: 600,
      margin: '0 0 14px',
      fontSize: 15,
      color: '#1e293b',
    },
    label: { display: 'block', margin: '10px 0' },
    lab: { fontSize: 12, color: '#475569', marginBottom: 5, fontWeight: 500 },
    required: { color: '#ef4444', marginLeft: 2 },
    input: {
      width: '100%',
      boxSizing: 'border-box' as const,
      padding: '9px 12px',
      borderRadius: 8,
      border: '1px solid #cbd5e1',
      background: '#fff',
      font: 'inherit',
      fontSize: 14,
    },
    actions: { display: 'flex', gap: 10, marginTop: 16, flexWrap: 'wrap' as const },
    btn: {
      padding: '8px 16px',
      border: '1px solid #cbd5e1',
      background: '#fff',
      borderRadius: 8,
      cursor: 'pointer',
      font: 'inherit',
      fontWeight: 500,
      fontSize: 13,
      color: '#334155',
    },
    btnSecondary: {
      padding: '7px 14px',
      border: '1px dashed #94a3b8',
      background: '#f8fafc',
      borderRadius: 8,
      cursor: 'pointer',
      font: 'inherit',
      fontWeight: 500,
      fontSize: 13,
      color: '#475569',
      width: '100%',
      marginTop: 4,
    },
    btnPrimary: {
      padding: '9px 20px',
      border: 'none',
      background: 'linear-gradient(135deg, #0f172a, #1e293b)',
      color: '#fff',
      borderRadius: 8,
      cursor: 'pointer',
      font: 'inherit',
      fontWeight: 600,
      fontSize: 14,
    },
    smallBtn: {
      padding: '5px 10px',
      fontSize: 11,
      background: '#e2e8f0',
      border: 'none',
      borderRadius: 6,
      cursor: 'pointer',
      fontWeight: 600,
      color: '#334155',
    },
    table: { borderCollapse: 'collapse' as const, width: '100%', fontSize: 13 },
    th: {
      borderBottom: '2px solid #e2e8f0',
      padding: 12,
      textAlign: 'left' as const,
      fontWeight: 600,
      color: '#475569',
      background: '#f8fafc',
    },
    td: {
      borderTop: '1px solid #f1f5f9',
      padding: 12,
      textAlign: 'left' as const,
      verticalAlign: 'top' as const,
      color: '#334155',
    },
    tr: { cursor: 'pointer', transition: 'background 0.1s' },
    pagination: {
      display: 'flex',
      gap: 8,
      marginTop: 16,
      justifyContent: 'center',
      alignItems: 'center',
      fontSize: 13,
      color: '#475569',
    },
    methodHeader: {
      marginBottom: 12,
      padding: 10,
      background: '#f8fafc',
      borderRadius: 8,
      borderLeft: '3px solid #3b82f6',
    },
    unit: { fontSize: 12, color: '#64748b', marginLeft: 6 },
    formula: {
      fontSize: 11,
      color: '#64748b',
      marginTop: 6,
      fontFamily: 'monospace',
      background: '#f1f5f9',
      padding: '4px 8px',
      borderRadius: 4,
      display: 'inline-block',
    },
    previewCard: {
      padding: 12,
      borderRadius: 8,
      marginTop: 12,
      fontSize: 13,
    },
    previewSuccess: { background: '#dcfce7', color: '#166534' },
    previewError: { background: '#fee2e2', color: '#991b1b' },
    previewNeutral: { background: '#f1f5f9', color: '#64748b' },
    skeleton: {
      background: '#f1f5f9',
      borderRadius: 4,
      animation: 'pulse 1.5s infinite',
    },
    loadingText: { textAlign: 'center' as const, color: '#64748b', padding: 20 },
  } as const;

  // === РЕНДЕР ===
  return (
    <div style={styles.container}>
      {/* HEADER */}
      <header style={styles.header}>
        <div style={styles.wrap}>
          <div style={styles.brand}>
            <div style={styles.logo}>Лаб</div>
            <div>
              <div style={{ fontWeight: 600 }}>Лаборатория дорожных материалов</div>
              <div style={styles.muted}>Система управления протоколами • ГОСТ 9128-2013</div>
            </div>
          </div>
          <div style={styles.row}>
            <span style={{
              ...styles.badge,
              ...(previewCompliance?.isCompliant === true ? styles.bGreen :
                  previewCompliance?.isCompliant === false ? styles.bRed : {})
            }}>
              {previewCompliance?.isCompliant === true ? '✓ Соответствует' :
               previewCompliance?.isCompliant === false ? '✗ Отклонение' :
               allInputsFilled ? '⏳ Проверка...' : '⏳ Ввод данных'}
            </span>
          </div>
        </div>
      </header>

      {/* MAIN CONTENT */}
      <main style={{ ...styles.wrap, padding: '24px', overflowY: 'auto', flex: 1 }}>
        <div style={styles.gridMain}>
          {/* LEFT: Лаборатория + Проба */}
          <section style={styles.card}>
            <h3 style={styles.title}>🏢 Лаборатория</h3>
            <label style={styles.label}>
              <div style={styles.lab}>Организация</div>
              <input style={styles.input} value={labName} onChange={e => setLabName(e.target.value)} />
            </label>
            <label style={styles.label}>
              <div style={styles.lab}>Оператор</div>
              <input style={styles.input} value={operator} onChange={e => setOperator(e.target.value)} />
            </label>
            <label style={styles.label}>
              <div style={styles.lab}>Проект</div>
              <input style={styles.input} value={project} onChange={e => setProject(e.target.value)} />
            </label>

            <h3 style={{ ...styles.title, marginTop: 20 }}>🧪 Проба</h3>
            <label style={styles.label}>
              <div style={styles.lab}>№ пробы *</div>
              <input style={styles.input} value={sampleNumber} onChange={e => setSampleNumber(e.target.value)} required />
            </label>
            <label style={styles.label}>
              <div style={styles.lab}>Место отбора *</div>
              <input style={styles.input} value={samplePlace} onChange={e => setSamplePlace(e.target.value)} required />
            </label>

            {/* Контекстные параметры (марка, зона и т.д.) */}
            {contextDimensions.length > 0 && (
              <>
                <h4 style={{ ...styles.title, marginTop: 16, fontSize: 14 }}>📐 Параметры пробы (ГОСТ)</h4>
                {contextDimensions.map((dim) => (
                  <label key={dim.id} style={styles.label}>
                    <div style={styles.lab}>{dim.label}</div>
                    {dim.data_type === 'select' && dim.possible_values?.length ? (
                      <select
                        style={styles.input}
                        value={sampleContext[dim.key_name] || ''}
                        onChange={(e) => setSampleContext({ ...sampleContext, [dim.key_name]: e.target.value })}
                      >
                        <option value="">— выберите —</option>
                        {dim.possible_values.map((val) => (
                          <option key={val} value={val}>{val}</option>
                        ))}
                      </select>
                    ) : (
                      <input
                        style={styles.input}
                        value={sampleContext[dim.key_name] || ''}
                        onChange={(e) => setSampleContext({ ...sampleContext, [dim.key_name]: e.target.value })}
                        placeholder={`Введите ${dim.label.toLowerCase()}`}
                      />
                    )}
                  </label>
                ))}
              </>
            )}

            <label style={styles.label}>
              <div style={styles.lab}>Примечание</div>
              <textarea
                style={{ ...styles.input, minHeight: 60, resize: 'vertical' as const }}
                value={sampleNote}
                onChange={e => setSampleNote(e.target.value)}
              />
            </label>

            {/* Группа */}
            <h3 style={{ ...styles.title, marginTop: 20 }}>📦 Группа</h3>
            <label style={styles.label}>
              <div style={styles.lab}>Выбрать группу</div>
              <select
                style={styles.input}
                value={selectedGroupId}
                onChange={(e) => {
                  const gid = e.target.value;
                  setSelectedGroupId(gid);
                  if (gid) {
                    const g = groups.find((grp) => grp.id === gid);
                    if (g) {
                      setSelectedMaterialId(g.material_id);
                      loadStandards(g.material_id);
                    }
                  } else {
                    setSelectedMaterialId('');
                    setStandards([]);
                    setMethods([]);
                    setContextDimensions([]);
                  }
                }}
              >
                <option value="">— без группы —</option>
                {groups.map((g) => (
                  <option key={g.id} value={g.id}>{g.name}</option>
                ))}
              </select>
            </label>
            <button style={styles.btnSecondary} onClick={() => setIsGroupModalOpen(true)}>
              + Создать группу
            </button>
          </section>

          {/* RIGHT: Стандарты + Методы */}
          <section style={styles.card}>
            <h3 style={styles.title}>📋 Стандарт / Метод</h3>
            
            <label style={styles.label}>
              <div style={styles.lab}>Материал *</div>
              <select
                style={styles.input}
                value={selectedMaterialId}
                disabled={!!selectedGroupId}
                onChange={(e) => {
                  setSelectedMaterialId(e.target.value);
                  loadStandards(e.target.value);
                }}
              >
                <option value="">— выберите —</option>
                {loading.materials ? (
                  <option disabled>Загрузка...</option>
                ) : materials.map((m) => (
                  <option key={m.id} value={m.id}>{m.name} {m.code ? `(${m.code})` : ''}</option>
                ))}
              </select>
            </label>

            <label style={styles.label}>
              <div style={styles.lab}>Стандарт *</div>
              <select
                style={styles.input}
                value={selectedStandardId}
                onChange={(e) => {
                  setSelectedStandardId(e.target.value);
                  loadMethods(e.target.value);
                }}
                disabled={!selectedMaterialId || loading.standards}
              >
                <option value="">— выберите —</option>
                {loading.standards ? (
                  <option disabled>Загрузка...</option>
                ) : standards.map((s) => (
                  <option key={s.id} value={s.id}>{s.name}</option>
                ))}
              </select>
            </label>

            <label style={styles.label}>
              <div style={styles.lab}>Метод *</div>
              <select
                style={styles.input}
                value={selectedMethodId}
                onChange={(e) => setSelectedMethodId(e.target.value)}
                disabled={!selectedStandardId || loading.methods}
              >
                <option value="">— выберите —</option>
                {loading.methods ? (
                  <option disabled>Загрузка...</option>
                ) : methods.map((m) => (
                  <option key={m.id} value={m.id}>{m.name} {m.unit ? `(${m.unit})` : ''}</option>
                ))}
              </select>
            </label>

            {/* Ввод данных метода */}
            <h3 style={{ ...styles.title, marginTop: 20 }}>📊 Результат</h3>
            <div style={{ marginBottom: 16 }}>
              {!currentMethod ? (
                <div style={styles.muted}>Выберите метод для ввода данных.</div>
              ) : (
                <div>
                  <div style={styles.methodHeader}>
                    <strong>{currentMethod.name}</strong>
                    {currentMethod.unit && <span style={styles.unit}>[{currentMethod.unit}]</span>}
                    {currentMethod.formula_expr && (
                      <div style={styles.formula}>Формула: {currentMethod.formula_expr}</div>
                    )}
                  </div>
                  
                  {(currentMethod.inputs || []).map((input) => (
                    <label key={input.param_key} style={styles.label}>
                      <div style={styles.lab}>
                        {input.label} {input.unit && `(${input.unit})`}
                        {input.is_required && <span style={styles.required}>*</span>}
                      </div>
                      {input.input_type === 'select' ? (
                        <select
                          style={styles.input}
                          value={rawInputs[input.param_key] || ''}
                          onChange={(e) => setRawInputs({ ...rawInputs, [input.param_key]: e.target.value })}
                        >
                          <option value="">— выберите —</option>
                        </select>
                      ) : (
                        <input
                          style={styles.input}
                          type="number"
                          step="any"
                          value={rawInputs[input.param_key] || ''}
                          onChange={(e) => setRawInputs({ ...rawInputs, [input.param_key]: e.target.value })}
                          placeholder="Введите значение"
                        />
                      )}
                    </label>
                  ))}
                </div>
              )}
            </div>

            {/* Превью соответствия */}
            {previewCompliance && previewCompliance.calculatedValue !== undefined && (
              <div style={{
                ...styles.previewCard,
                ...(previewCompliance.isCompliant === true ? styles.previewSuccess :
                    previewCompliance.isCompliant === false ? styles.previewError : styles.previewNeutral)
              }}>
                <strong>🔍 Предпросмотр:</strong><br />
                Результат: <strong>{previewCompliance.calculatedValue}</strong> {currentMethod?.unit}<br />
                Норма: {previewCompliance.norm}<br />
                Статус: {
                  previewCompliance.isCompliant === true ? '✓ Соответствует' :
                  previewCompliance.isCompliant === false ? '✗ Отклонение' :
                  '—'
                }
                {previewCompliance.deviation && (
                  <div style={{ fontSize: 11, marginTop: 4 }}>{previewCompliance.deviation}</div>
                )}
              </div>
            )}

            <div style={styles.actions}>
              <button style={styles.btn} onClick={() => {
                setRawInputs({});
                setSampleContext({});
                setSelectedMethodId('');
              }}>↺ Сброс</button>
            </div>
          </section>
        </div>

        {/* SAVE BUTTON */}
        <section style={{ ...styles.card, marginTop: 24 }}>
          <div style={styles.row}>
            <h3 style={styles.title}>✅ Сравнение с нормой</h3>
            <button
              style={{ ...styles.btnPrimary, opacity: loading.save ? 0.7 : 1 }}
              disabled={loading.save || !canSave}
              onClick={handleSaveProtocol}
            >
              {loading.save ? '⏳ Сохранение...' : '💾 Сохранить протокол'}
            </button>
          </div>
          <div style={{
            marginTop: 12,
            padding: 12,
            background: canSave
              ? (previewCompliance?.isCompliant === false ? '#fee2e2' : '#dcfce7')
              : '#f1f5f9',
            borderRadius: 8,
            fontSize: 14,
          }}>
            {currentMethod ? (
              canSave ? (
                previewCompliance?.isCompliant === false
                  ? <span style={{ color: '#991b1b' }}>✗ Отклонение от нормы</span>
                  : <span style={{ color: '#166534' }}>✓ Готово к сохранению</span>
              ) : (
                <span style={{ color: '#64748b' }}>⏳ Заполните обязательные параметры</span>
              )
            ) : <span style={{ color: '#64748b' }}>👈 Выберите метод</span>}
          </div>
        </section>

        {/* HISTORY TABLES */}
        <div style={styles.gridHistory}>
          {/* ПРОТОКОЛЫ */}
          <section style={styles.card}>
            <h3 style={styles.title}>📄 История протоколов</h3>
            {loading.protocols ? (
              <div style={styles.loadingText}>Загрузка...</div>
            ) : (
              <div style={{ overflowX: 'auto' }}>
                <table style={styles.table}>
                  <thead>
                    <tr>
                      <th style={styles.th}>Дата</th>
                      <th style={styles.th}>№</th>
                      <th style={styles.th}>Материал</th>
                      <th style={styles.th}>Лаборатория</th>
                      <th style={styles.th}>Действия</th>
                    </tr>
                  </thead>
                  <tbody>
                    {protocols.length === 0 ? (
                      <tr>
                        <td colSpan={5} style={{ textAlign: 'center', padding: 24, color: '#64748b' }}>
                          Нет протоколов
                        </td>
                      </tr>
                    ) : protocols.map((p) => (
                      <tr
                        key={p.id}
                        style={styles.tr}
                        onClick={() => handleViewProtocol(p.id)}
                        onMouseEnter={(e) => e.currentTarget.style.background = '#f8fafc'}
                        onMouseLeave={(e) => e.currentTarget.style.background = ''}
                      >
                        <td style={styles.td}>{formatDate(p.created_at)}</td>
                        <td style={styles.td}>{p.sample?.sample_number || '—'}</td>
                        <td style={styles.td}>{p.material_name || '—'}</td>
                        <td style={styles.td}>{p.lab_name || '—'}</td>
                        <td style={styles.td}>
                          <button
                            style={styles.smallBtn}
                            onClick={(e) => { e.stopPropagation(); handleDownloadProtocolPDF(p.id); }}
                          >
                            📄 PDF
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
            {protocolsMeta && protocolsMeta.total_pages > 1 && (
              <div style={styles.pagination}>
                <button
                  style={styles.btn}
                  disabled={protocolsPage === 1 || loading.protocols}
                  onClick={() => loadProtocolsList(protocolsPage - 1)}
                >←</button>
                <span>{protocolsPage} / {protocolsMeta.total_pages}</span>
                <button
                  style={styles.btn}
                  disabled={protocolsPage === protocolsMeta.total_pages || loading.protocols}
                  onClick={() => loadProtocolsList(protocolsPage + 1)}
                >→</button>
              </div>
            )}
          </section>

          {/* ГРУППЫ */}
          <section style={styles.card}>
            <h3 style={styles.title}>📦 Комплексные испытания</h3>
            {loading.groups ? (
              <div style={styles.loadingText}>Загрузка...</div>
            ) : (
              <div style={{ overflowX: 'auto' }}>
                <table style={styles.table}>
                  <thead>
                    <tr>
                      <th style={styles.th}>Название</th>
                      <th style={styles.th}>Материал</th>
                      <th style={styles.th}>Проект</th>
                      <th style={styles.th}>Действия</th>
                    </tr>
                  </thead>
                  <tbody>
                    {groups.length === 0 ? (
                      <tr>
                        <td colSpan={4} style={{ textAlign: 'center', padding: 24, color: '#64748b' }}>
                          Нет групп
                        </td>
                      </tr>
                    ) : groups.map((g) => {
                      const matName = materials.find((m) => m.id === g.material_id)?.name || '—';
                      return (
                        <tr key={g.id} style={styles.tr}>
                          <td style={styles.td}><strong>{g.name}</strong></td>
                          <td style={styles.td}>{matName}</td>
                          <td style={styles.td}>{g.project_name || '—'}</td>
                          <td style={styles.td}>
                            <button
                              style={styles.smallBtn}
                              onClick={async (e) => {
                                e.stopPropagation();
                                await handleViewGroupSummary(g.id);
                              }}
                            >
                              📊 Сводка
                            </button>
                            <button
                              style={{ ...styles.smallBtn, marginLeft: 8 }}
                              onClick={(e) => { e.stopPropagation(); handleDownloadGroupPDF(g.id); }}
                            >
                              📄 PDF
                            </button>
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            )}
            {groupsMeta && groupsMeta.total_pages > 1 && (
              <div style={styles.pagination}>
                <button
                  style={styles.btn}
                  disabled={groupsPage === 1 || loading.groups}
                  onClick={() => loadGroupsList(groupsPage - 1)}
                >←</button>
                <span>{groupsPage} / {groupsMeta.total_pages}</span>
                <button
                  style={styles.btn}
                  disabled={groupsPage === groupsMeta.total_pages || loading.groups}
                  onClick={() => loadGroupsList(groupsPage + 1)}
                >→</button>
              </div>
            )}
          </section>
        </div>
      </main>

      {/* MODALS */}
      
      {/* Модалка создания группы */}
      <Modal
        isOpen={isGroupModalOpen}
        onClose={() => setIsGroupModalOpen(false)}
        title="✨ Новая группа"
        size="md"
      >
        <label style={styles.label}>
          <div style={styles.lab}>Название *</div>
          <input
            style={styles.input}
            value={newGroup.name}
            onChange={(e) => setNewGroup({ ...newGroup, name: e.target.value })}
            placeholder="Введите название группы"
          />
        </label>
        <label style={styles.label}>
          <div style={styles.lab}>Материал *</div>
          <select
            style={styles.input}
            value={newGroup.materialId}
            onChange={(e) => setNewGroup({ ...newGroup, materialId: e.target.value })}
          >
            <option value="">— выберите —</option>
            {materials.map((m) => (
              <option key={m.id} value={m.id}>{m.name}</option>
            ))}
          </select>
        </label>
        <label style={styles.label}>
          <div style={styles.lab}>Проект</div>
          <input
            style={styles.input}
            value={newGroup.project}
            onChange={(e) => setNewGroup({ ...newGroup, project: e.target.value })}
            placeholder="Название проекта"
          />
        </label>
        <label style={styles.label}>
          <div style={styles.lab}>Локация</div>
          <input
            style={styles.input}
            value={newGroup.location}
            onChange={(e) => setNewGroup({ ...newGroup, location: e.target.value })}
            placeholder="Место проведения"
          />
        </label>
        <div style={styles.actions}>
          <button style={styles.btn} onClick={() => setIsGroupModalOpen(false)}>Отмена</button>
          <button
            style={styles.btnPrimary}
            onClick={handleCreateGroup}
            disabled={loading.save || !newGroup.name || !newGroup.materialId}
          >
            {loading.save ? '⏳ Создание...' : 'Создать'}
          </button>
        </div>
      </Modal>

      {/* Модалка просмотра протокола */}
      {viewingProtocol && (
        <Modal
          isOpen={!!viewingProtocol}
          onClose={() => setViewingProtocol(null)}
          title={`📄 Протокол №${viewingProtocol.sample?.sample_number || '—'}`}
          size="lg"
        >
          <div style={{
            marginBottom: 16,
            fontSize: 13,
            background: '#f8fafc',
            padding: 16,
            borderRadius: 8,
            display: 'grid',
            gap: 8,
          }}>
            <div><strong>📅 Дата:</strong> {formatDate(viewingProtocol.created_at)}</div>
            <div><strong>🧱 Материал:</strong> {materials.find((m) => m.id === viewingProtocol.sample?.material_id)?.name || '—'}</div>
            <div><strong>🏢 Лаборатория:</strong> {viewingProtocol.lab_name || '—'}</div>
            <div><strong>👤 Оператор:</strong> {viewingProtocol.operator_name || '—'}</div>
            <div><strong>📦 Статус:</strong> {viewingProtocol.status}</div>
          </div>

          {/* Результаты */}
          {viewingProtocol.results?.length ? (
            <div style={{ marginBottom: 16 }}>
              <h4 style={{ ...styles.title, fontSize: 14 }}>Результаты испытаний</h4>
              <table style={styles.table}>
                <thead>
                  <tr>
                    <th style={styles.th}>Метод</th>
                    <th style={styles.th}>Значение</th>
                    <th style={styles.th}>Норма</th>
                    <th style={styles.th}>Статус</th>
                  </tr>
                </thead>
                <tbody>
                  {viewingProtocol.results.map((r) => {
                    const norm = (r.min_norm != null && r.max_norm != null)
                      ? `${r.min_norm} – ${r.max_norm}`
                      : (r.min_norm != null) ? `≥ ${r.min_norm}`
                      : (r.max_norm != null) ? `≤ ${r.max_norm}` : '—';
                    return (
                      <tr key={r.id}>
                        <td style={styles.td}>{r.method_name || r.method_id}</td>
                        <td style={styles.td}>{formatNumber(r.calculated_value, r.method_unit)}</td>
                        <td style={styles.td}>{norm}</td>
                        <td style={styles.td}>
                          {r.is_compliant === true
                            ? <span style={styles.bGreen}>✓</span>
                            : r.is_compliant === false
                            ? <span style={styles.bRed}>✗</span>
                            : '—'}
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          ) : (
            <div style={styles.muted}>Нет результатов</div>
          )}

          <div style={{ ...styles.actions, justifyContent: 'flex-end' }}>
            <button style={styles.btnPrimary} onClick={() => handleDownloadProtocolPDF(viewingProtocol.id)}>
              📥 Скачать PDF
            </button>
          </div>
        </Modal>
      )}

      {/* Модалка сводки по группе */}
      {viewingSummary && (
        <Modal
          isOpen={!!viewingSummary}
          onClose={() => setViewingSummary(null)}
          title={`📊 Сводка: ${viewingSummary.group_name}`}
          size="xl"
        >
          <div style={{
            marginBottom: 16,
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))',
            gap: 12,
            background: '#f8fafc',
            padding: 16,
            borderRadius: 8,
          }}>
            <div><strong>🧱 Материал:</strong><br />{materials.find((m) => m.id === viewingSummary.material_id)?.name || '—'}</div>
            <div><strong>🧪 Проб:</strong><br />{viewingSummary.total_samples}</div>
            <div><strong>✅ Соответствие:</strong><br />{viewingSummary.compliant_rate.toFixed(1)}%</div>
          </div>

          <table style={styles.table}>
            <thead>
              <tr>
                <th style={styles.th}>Метод</th>
                <th style={styles.th}>Норма</th>
                <th style={styles.th}>Статус</th>
                <th style={styles.th}>Значения по пробам</th>
              </tr>
            </thead>
            <tbody>
              {viewingSummary.results.map((r, idx) => {
                const norm = (r.min_value != null && r.max_value != null)
                  ? `${r.min_value} – ${r.max_value}`
                  : (r.min_value != null) ? `≥ ${r.min_value}`
                  : (r.max_value != null) ? `≤ ${r.max_value}` : '—';
                return (
                  <tr key={idx} style={styles.tr}>
                    <td style={styles.td}>
                      <strong>{r.method_name}</strong><br />
                      <span style={{ fontSize: 11, color: '#64748b' }}>{r.unit}</span>
                    </td>
                    <td style={styles.td}>{norm}</td>
                    <td style={styles.td}>
                      {r.is_compliant
                        ? <span style={styles.bGreen}>✓</span>
                        : <span style={styles.bRed}>✗</span>}
                    </td>
                    <td style={styles.td}>
                      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4 }}>
                        {r.trials.slice(0, 5).map((t, i) => (
                          <span
                            key={i}
                            title={`Проба ${t.sample_number}: ${t.value}${t.deviation ? `\n${t.deviation}` : ''}`}
                            style={{
                              fontSize: 11,
                              padding: '3px 8px',
                              borderRadius: 4,
                              background: t.is_compliant ? '#dcfce7' : '#fee2e2',
                              color: t.is_compliant ? '#166534' : '#991b1b',
                            }}
                          >
                            {t.sample_number}: {t.value}
                          </span>
                        ))}
                        {r.trials.length > 5 && (
                          <span style={{ fontSize: 11, color: '#64748b' }}>
                            +{r.trials.length - 5} ещё
                          </span>
                        )}
                      </div>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>

          <div style={{ ...styles.actions, marginTop: 16, justifyContent: 'flex-end' }}>
            <button style={styles.btnPrimary} onClick={() => handleDownloadGroupPDF(viewingSummary.group_id)}>
              📥 Скачать отчёт (PDF)
            </button>
          </div>
        </Modal>
      )}

      {/* TOAST */}
      {toast && (
        <Toast
          message={toast.msg}
          error={toast.error}
          onClose={() => setToast(null)}
        />
      )}
    </div>
  );
}