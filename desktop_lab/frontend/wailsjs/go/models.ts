export namespace models {
	
	export class ContextDimension {
	    id: string;
	    standard_id: string;
	    key_name: string;
	    label: string;
	    data_type: string;
	    possible_values: string[];
	
	    static createFrom(source: any = {}) {
	        return new ContextDimension(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.standard_id = source["standard_id"];
	        this.key_name = source["key_name"];
	        this.label = source["label"];
	        this.data_type = source["data_type"];
	        this.possible_values = source["possible_values"];
	    }
	}
	export class CreateResultDTO {
	    method_id: string;
	    raw_inputs: Record<string, string>;
	    note?: string;
	
	    static createFrom(source: any = {}) {
	        return new CreateResultDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.method_id = source["method_id"];
	        this.raw_inputs = source["raw_inputs"];
	        this.note = source["note"];
	    }
	}
	export class CreateSampleDTO {
	    sample_number: string;
	    material_id: string;
	    // Go type: time
	    collection_date?: any;
	    context_params: Record<string, string>;
	    note?: string;
	
	    static createFrom(source: any = {}) {
	        return new CreateSampleDTO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sample_number = source["sample_number"];
	        this.material_id = source["material_id"];
	        this.collection_date = this.convertValues(source["collection_date"], null);
	        this.context_params = source["context_params"];
	        this.note = source["note"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CreateProtocolRequest {
	    group_id: string;
	    sample: CreateSampleDTO;
	    lab_name: string;
	    operator_name: string;
	    results: CreateResultDTO[];
	
	    static createFrom(source: any = {}) {
	        return new CreateProtocolRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.group_id = source["group_id"];
	        this.sample = this.convertValues(source["sample"], CreateSampleDTO);
	        this.lab_name = source["lab_name"];
	        this.operator_name = source["operator_name"];
	        this.results = this.convertValues(source["results"], CreateResultDTO);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class ExperimentGroup {
	    id: string;
	    name: string;
	    material_id: string;
	    project_name: string;
	    location?: string;
	    // Go type: time
	    created_at: any;
	    material_name?: string;
	    sample_count?: number;
	
	    static createFrom(source: any = {}) {
	        return new ExperimentGroup(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.material_id = source["material_id"];
	        this.project_name = source["project_name"];
	        this.location = source["location"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.material_name = source["material_name"];
	        this.sample_count = source["sample_count"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class MethodTrial {
	    protocol_id: string;
	    sample_number: string;
	    value: number;
	    is_compliant: boolean;
	    deviation?: string;
	
	    static createFrom(source: any = {}) {
	        return new MethodTrial(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.protocol_id = source["protocol_id"];
	        this.sample_number = source["sample_number"];
	        this.value = source["value"];
	        this.is_compliant = source["is_compliant"];
	        this.deviation = source["deviation"];
	    }
	}
	export class MethodResultSummary {
	    method_id: string;
	    method_name: string;
	    unit: string;
	    min_value?: number;
	    max_value?: number;
	    is_compliant: boolean;
	    trials: MethodTrial[];
	
	    static createFrom(source: any = {}) {
	        return new MethodResultSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.method_id = source["method_id"];
	        this.method_name = source["method_name"];
	        this.unit = source["unit"];
	        this.min_value = source["min_value"];
	        this.max_value = source["max_value"];
	        this.is_compliant = source["is_compliant"];
	        this.trials = this.convertValues(source["trials"], MethodTrial);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class GroupSummary {
	    group_id: string;
	    group_name: string;
	    material_id: string;
	    total_samples: number;
	    compliant_rate: number;
	    results: MethodResultSummary[];
	
	    static createFrom(source: any = {}) {
	        return new GroupSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.group_id = source["group_id"];
	        this.group_name = source["group_name"];
	        this.material_id = source["material_id"];
	        this.total_samples = source["total_samples"];
	        this.compliant_rate = source["compliant_rate"];
	        this.results = this.convertValues(source["results"], MethodResultSummary);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class LimitCondition {
	    id: string;
	    limit_id: string;
	    dimension_key: string;
	    condition_operator: string;
	    expected_value: string;
	
	    static createFrom(source: any = {}) {
	        return new LimitCondition(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.limit_id = source["limit_id"];
	        this.dimension_key = source["dimension_key"];
	        this.condition_operator = source["condition_operator"];
	        this.expected_value = source["expected_value"];
	    }
	}
	export class Material {
	    id: string;
	    name: string;
	    code?: string;
	    // Go type: time
	    created_at: any;
	
	    static createFrom(source: any = {}) {
	        return new Material(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.code = source["code"];
	        this.created_at = this.convertValues(source["created_at"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class MethodInput {
	    id: string;
	    method_id: string;
	    param_key: string;
	    label: string;
	    unit?: string;
	    input_type: string;
	    is_required: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MethodInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.method_id = source["method_id"];
	        this.param_key = source["param_key"];
	        this.label = source["label"];
	        this.unit = source["unit"];
	        this.input_type = source["input_type"];
	        this.is_required = source["is_required"];
	    }
	}
	
	
	export class NormativeLimit {
	    id: string;
	    method_id: string;
	    limit_type: string;
	    min_value?: number;
	    max_value?: number;
	    discrete_values?: string[];
	    note?: string;
	    priority: number;
	    conditions?: LimitCondition[];
	
	    static createFrom(source: any = {}) {
	        return new NormativeLimit(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.method_id = source["method_id"];
	        this.limit_type = source["limit_type"];
	        this.min_value = source["min_value"];
	        this.max_value = source["max_value"];
	        this.discrete_values = source["discrete_values"];
	        this.note = source["note"];
	        this.priority = source["priority"];
	        this.conditions = this.convertValues(source["conditions"], LimitCondition);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PaginatedMetadata {
	    total: number;
	    page: number;
	    page_size: number;
	    total_pages: number;
	
	    static createFrom(source: any = {}) {
	        return new PaginatedMetadata(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.page = source["page"];
	        this.page_size = source["page_size"];
	        this.total_pages = source["total_pages"];
	    }
	}
	export class TestResult {
	    id: string;
	    protocol_id: string;
	    method_id: string;
	    input_data: Record<string, any>;
	    calculated_value?: number;
	    applied_limit_id?: string;
	    is_compliant?: boolean;
	    deviation_msg?: string;
	    note?: string;
	    // Go type: time
	    created_at: any;
	    method_name?: string;
	    method_unit?: string;
	    min_norm?: number;
	    max_norm?: number;
	
	    static createFrom(source: any = {}) {
	        return new TestResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.protocol_id = source["protocol_id"];
	        this.method_id = source["method_id"];
	        this.input_data = source["input_data"];
	        this.calculated_value = source["calculated_value"];
	        this.applied_limit_id = source["applied_limit_id"];
	        this.is_compliant = source["is_compliant"];
	        this.deviation_msg = source["deviation_msg"];
	        this.note = source["note"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.method_name = source["method_name"];
	        this.method_unit = source["method_unit"];
	        this.min_norm = source["min_norm"];
	        this.max_norm = source["max_norm"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Sample {
	    id: string;
	    group_id: string;
	    material_id: string;
	    sample_number: string;
	    // Go type: time
	    collection_date?: any;
	    context_params: Record<string, string>;
	    note?: string;
	    // Go type: time
	    created_at: any;
	
	    static createFrom(source: any = {}) {
	        return new Sample(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.group_id = source["group_id"];
	        this.material_id = source["material_id"];
	        this.sample_number = source["sample_number"];
	        this.collection_date = this.convertValues(source["collection_date"], null);
	        this.context_params = source["context_params"];
	        this.note = source["note"];
	        this.created_at = this.convertValues(source["created_at"], null);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ProtocolListItem {
	    id: string;
	    sample_id: string;
	    protocol_number?: string;
	    lab_name?: string;
	    operator_name?: string;
	    // Go type: time
	    test_date?: any;
	    status: string;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	    // Go type: Sample
	    sample?: any;
	    results?: TestResult[];
	    material_name: string;
	
	    static createFrom(source: any = {}) {
	        return new ProtocolListItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.sample_id = source["sample_id"];
	        this.protocol_number = source["protocol_number"];
	        this.lab_name = source["lab_name"];
	        this.operator_name = source["operator_name"];
	        this.test_date = this.convertValues(source["test_date"], null);
	        this.status = source["status"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
	        this.sample = this.convertValues(source["sample"], null);
	        this.results = this.convertValues(source["results"], TestResult);
	        this.material_name = source["material_name"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ProtocolListResponse {
	    items: ProtocolListItem[];
	    meta: PaginatedMetadata;
	
	    static createFrom(source: any = {}) {
	        return new ProtocolListResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], ProtocolListItem);
	        this.meta = this.convertValues(source["meta"], PaginatedMetadata);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TestMethod {
	    id: string;
	    standard_id: string;
	    code?: string;
	    name: string;
	    description?: string;
	    formula_expr?: string;
	    unit: string;
	    result_type: string;
	    is_mandatory: boolean;
	    inputs?: MethodInput[];
	    limits?: NormativeLimit[];
	
	    static createFrom(source: any = {}) {
	        return new TestMethod(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.standard_id = source["standard_id"];
	        this.code = source["code"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.formula_expr = source["formula_expr"];
	        this.unit = source["unit"];
	        this.result_type = source["result_type"];
	        this.is_mandatory = source["is_mandatory"];
	        this.inputs = this.convertValues(source["inputs"], MethodInput);
	        this.limits = this.convertValues(source["limits"], NormativeLimit);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Standard {
	    id: string;
	    material_id: string;
	    name: string;
	    description?: string;
	    // Go type: time
	    valid_from?: any;
	    // Go type: time
	    valid_to?: any;
	    dimensions?: ContextDimension[];
	    methods?: TestMethod[];
	
	    static createFrom(source: any = {}) {
	        return new Standard(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.material_id = source["material_id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.valid_from = this.convertValues(source["valid_from"], null);
	        this.valid_to = this.convertValues(source["valid_to"], null);
	        this.dimensions = this.convertValues(source["dimensions"], ContextDimension);
	        this.methods = this.convertValues(source["methods"], TestMethod);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

