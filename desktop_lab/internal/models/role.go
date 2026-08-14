package models

// Role представляет роль пользователя в системе
type Role string

const (
	RoleKey string = "role"
	// RoleAdmin - администратор системы (полный доступ)
	RoleAdmin Role = "admin"
	// RoleEngineer - инженер (создание и редактирование протоколов, стандартов)
	RoleEngineer Role = "engineer"
	// RoleTechnician - техник (создание проб, выполнение тестов)
	RoleTechnician Role = "technician"
	// RoleClient - клиент (только просмотр своих протоколов)
	RoleClient Role = "client"
)

// IsValid проверяет, является ли роль допустимой
func (r Role) IsValid() bool {
	switch r {
	case RoleAdmin, RoleEngineer, RoleTechnician, RoleClient:
		return true
	default:
		return false
	}
}

// String возвращает строковое представление роли
func (r Role) String() string {
	return string(r)
}

// Permission представляет разрешение на выполнение действия
type Permission string

const (
	// Пользователи
	PermUserRead   Permission = "user:read"
	PermUserCreate Permission = "user:create"
	PermUserUpdate Permission = "user:update"
	PermUserDelete Permission = "user:delete"

	// Протоколы
	PermProtocolRead   Permission = "protocol:read"
	PermProtocolCreate Permission = "protocol:create"
	PermProtocolUpdate Permission = "protocol:update"
	PermProtocolDelete Permission = "protocol:delete"

	// Стандарты
	PermStandardRead   Permission = "standard:read"
	PermStandardCreate Permission = "standard:create"
	PermStandardUpdate Permission = "standard:update"
	PermStandardDelete Permission = "standard:delete"

	// Проба
	PermSampleRead   Permission = "sample:read"
	PermSampleCreate Permission = "sample:create"
	PermSampleUpdate Permission = "sample:update"
	PermSampleDelete Permission = "sample:delete"

	// Группы экспериментов
	PermGroupRead   Permission = "group:read"
	PermGroupCreate Permission = "group:create"
	PermGroupUpdate Permission = "group:update"
	PermGroupDelete Permission = "group:delete"

	// Материалы
	PermMaterialRead   Permission = "material:read"
	PermMaterialCreate Permission = "material:create"
	PermMaterialUpdate Permission = "material:update"
	PermMaterialDelete Permission = "material:delete"

	// Измерения
	PermDimensionRead   Permission = "dimension:read"
	PermDimensionCreate Permission = "dimension:create"
	PermDimensionUpdate Permission = "dimension:update"
	PermDimensionDelete Permission = "dimension:delete"

	// Отчеты
	PermReportRead   Permission = "report:read"
	PermReportCreate Permission = "report:create"

	// Организация
	PermOrganizationRead   Permission = "organization:read"
	PermOrganizationCreate Permission = "organization:create"
	PermOrganizationUpdate Permission = "organization:update"
	PermOrganizationDelete Permission = "organization:delete"
)

// RolePermissions определяет разрешения для каждой роли
var RolePermissions = map[Role][]Permission{
	RoleAdmin: {
		// Полный доступ ко всем ресурсам
		PermUserRead, PermUserCreate, PermUserUpdate, PermUserDelete,
		PermProtocolRead, PermProtocolCreate, PermProtocolUpdate, PermProtocolDelete,
		PermStandardRead, PermStandardCreate, PermStandardUpdate, PermStandardDelete,
		PermSampleRead, PermSampleCreate, PermSampleUpdate, PermSampleDelete,
		PermGroupRead, PermGroupCreate, PermGroupUpdate, PermGroupDelete,
		PermMaterialRead, PermMaterialCreate, PermMaterialUpdate, PermMaterialDelete,
		PermDimensionRead, PermDimensionCreate, PermDimensionUpdate, PermDimensionDelete,
		PermReportRead, PermReportCreate,
		PermOrganizationRead, PermOrganizationCreate, PermOrganizationUpdate, PermOrganizationDelete,
	},
	RoleEngineer: {
		// Чтение всех пользователей
		PermUserRead,
		// Полный доступ к протоколам, стандартам, отчетам
		PermProtocolRead, PermProtocolCreate, PermProtocolUpdate, PermProtocolDelete,
		PermStandardRead, PermStandardCreate, PermStandardUpdate, PermStandardDelete,
		PermReportRead, PermReportCreate,
		// Чтение материалов и измерений
		PermMaterialRead, PermDimensionRead,
		// Доступ к группам и пробам
		PermGroupRead, PermGroupCreate, PermGroupUpdate, PermGroupDelete,
		PermSampleRead, PermSampleCreate, PermSampleUpdate, PermSampleDelete,
		// Чтение организации
		PermOrganizationRead,
	},
	RoleTechnician: {
		// Чтение пользователей
		PermUserRead,
		// Чтение и создание протоколов (но не удаление)
		PermProtocolRead, PermProtocolCreate,
		// Чтение стандартов
		PermStandardRead,
		// Полный доступ к пробам
		PermSampleRead, PermSampleCreate, PermSampleUpdate,
		// Чтение и создание групп
		PermGroupRead, PermGroupCreate,
		// Чтение материалов и измерений
		PermMaterialRead, PermDimensionRead,
		// Чтение отчетов
		PermReportRead,
		// Чтение организации
		PermOrganizationRead,
	},
	RoleClient: {
		// Только чтение своих протоколов и отчетов
		PermProtocolRead,
		PermReportRead,
		// Чтение своей организации
		PermOrganizationRead,
	},
}

// HasPermission проверяет, есть ли у роли определенное разрешение
func (r Role) HasPermission(permission Permission) bool {
	permissions, exists := RolePermissions[r]
	if !exists {
		return false
	}

	for _, perm := range permissions {
		if perm == permission {
			return true
		}
	}

	return false
}

// HasAnyPermission проверяет, есть ли у роли хотя бы одно из указанных разрешений
func (r Role) HasAnyPermission(permissions ...Permission) bool {
	for _, perm := range permissions {
		if r.HasPermission(perm) {
			return true
		}
	}
	return false
}

// HasAllPermissions проверяет, есть ли у роли все указанные разрешения
func (r Role) HasAllPermissions(permissions ...Permission) bool {
	for _, perm := range permissions {
		if !r.HasPermission(perm) {
			return false
		}
	}
	return true
}

// GetAvailablePermissions возвращает все разрешения для роли
func (r Role) GetAvailablePermissions() []Permission {
	if perms, exists := RolePermissions[r]; exists {
		return perms
	}
	return []Permission{}
}
