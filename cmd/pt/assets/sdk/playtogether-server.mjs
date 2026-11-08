import { createHash, createPrivateKey, randomBytes, sign, verify } from "node:crypto";
import { readFileSync } from "node:fs";
import { createServer } from "node:http";
// ../../node_modules/zod/v4/core/util.js
function getEnumValues(entries) {
	const numericValues = Object.values(entries).filter((v) => typeof v === "number");
	return Object.entries(entries).filter(([k, _]) => numericValues.indexOf(+k) === -1).map(([_, v]) => v);
}
function jsonStringifyReplacer(_, value) {
	if (typeof value === "bigint") return value.toString();
	return value;
}
var Cached = class {
	constructor(getter) {
		this._getter = getter;
		this._value = void 0;
	}
	get value() {
		const getter = this._getter;
		if (getter !== void 0) {
			this._value = getter();
			this._getter = void 0;
		}
		return this._value;
	}
};
function cached(getter) {
	return new Cached(getter);
}
function nullish(input) {
	return input === null || input === void 0;
}
function cleanRegex(source) {
	const start = source.startsWith("^") ? 1 : 0;
	const end = source.endsWith("$") ? source.length - 1 : source.length;
	return source.slice(start, end);
}
function assignProp(target, prop, value) {
	Object.defineProperty(target, prop, {
		value,
		writable: true,
		enumerable: true,
		configurable: true
	});
}
var captureStackTrace = "captureStackTrace" in Error ? Error.captureStackTrace : (..._args) => {};
function isObject(data) {
	return typeof data === "object" && data !== null && !Array.isArray(data);
}
function isPlainObject(o) {
	if (isObject(o) === false) return false;
	const ctor = o.constructor;
	if (ctor === void 0) return true;
	if (typeof ctor !== "function") return true;
	const prot = ctor.prototype;
	if (isObject(prot) === false) return false;
	if (Object.prototype.hasOwnProperty.call(prot, "isPrototypeOf") === false) return false;
	return true;
}
function shallowClone(o) {
	if (isPlainObject(o)) return { ...o };
	if (Array.isArray(o)) return [...o];
	if (o instanceof Map) return new Map(o);
	if (o instanceof Set) return new Set(o);
	return o;
}
var propertyKeyTypes = /* @__PURE__*/ new Set([
	"string",
	"number",
	"symbol"
]);
function escapeRegex(str) {
	return str.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}
function clone(inst, def, params) {
	const cl = new inst._zod.constr(def ?? inst._zod.def);
	if (!def || params?.parent) cl._zod.parent = inst;
	return cl;
}
function normalizeParams(_params) {
	const params = _params;
	if (!params) return {};
	if (typeof params === "string") return { error: () => params };
	if (params?.message !== void 0) {
		if (params?.error !== void 0) throw new Error("Cannot specify both `message` and `error` params");
		params.error = params.message;
	}
	delete params.message;
	if (typeof params.error === "string") return {
		...params,
		error: () => params.error
	};
	return params;
}
function optionalKeys(shape) {
	return Object.keys(shape).filter((k) => {
		return shape[k]._zod.optin !== void 0 && shape[k]._zod.optout === "optional";
	});
}
var NUMBER_FORMAT_RANGES = /*@__PURE__*/ (() => ({
	safeint: [Number.MIN_SAFE_INTEGER, Number.MAX_SAFE_INTEGER],
	int32: [-2147483648, 2147483647],
	uint32: [0, 4294967295],
	float32: [-34028234663852886e22, 34028234663852886e22],
	float64: [-Number.MAX_VALUE, Number.MAX_VALUE]
}))();
function aborted(x, startIndex = 0) {
	if (x.aborted === true) return true;
	for (let i = startIndex; i < x.issues.length; i++) if (x.issues[i]?.continue !== true) return true;
	return false;
}
function explicitlyAborted(x, startIndex = 0) {
	if (x.aborted === true) return true;
	for (let i = startIndex; i < x.issues.length; i++) if (x.issues[i]?.continue === false) return true;
	return false;
}
function prefixIssues(path, issues) {
	return issues.map((iss) => {
		var _a;
		(_a = iss).path ?? (_a.path = []);
		iss.path.unshift(path);
		return iss;
	});
}
function unwrapMessage(message) {
	return typeof message === "string" ? message : message?.message;
}
/** The leading size checks of `defChecks` that set an upper bound, each paired with it. */
function sizeCaps(defChecks) {
	const caps = [];
	for (const ch of defChecks ?? []) {
		const d = ch._zod.def;
		if (!/_(length|size)$|_equals$/.test(d.check)) break;
		const max = d.maximum ?? d.length ?? d.size;
		if (max >= 0) caps.push([max, ch]);
	}
	return caps;
}
function attachSchema(issues, start, inst) {
	var _a;
	for (let i = start; i < issues.length; i++) (_a = issues[i]).schema ?? (_a.schema = inst);
}
function finalizeIssue(iss, ctx, config) {
	var _a;
	const traits = iss.inst?._zod?.traits;
	if (traits?.has("$ZodType")) {
		if (traits.has("$ZodCheck")) (_a = iss).schema ?? (_a.schema = iss.inst);
		else iss.schema = iss.inst;
	}
	const schemaError = iss.schema !== iss.inst ? iss.schema?._zod.def?.error : void 0;
	const message = iss.message ? iss.message : unwrapMessage(iss.inst?._zod.def?.error?.(iss)) ?? unwrapMessage(schemaError?.(iss)) ?? unwrapMessage(ctx?.error?.(iss)) ?? unwrapMessage(config.customError?.(iss)) ?? unwrapMessage(config.localeError?.(iss)) ?? "Invalid input";
	const full = {};
	for (const k of Object.keys(iss)) {
		if (k === "inst" || k === "schema" || k === "continue" || k === "input" || k === "__proto__") continue;
		full[k] = iss[k];
	}
	full.path ?? (full.path = []);
	full.message = message;
	if (ctx?.reportInput) full.input = iss.input;
	return full;
}
var highSurrogate = /[\uD800-\uDBFF]/;
function codePointLength(str) {
	const units = str.length;
	if (!highSurrogate.test(str)) return units;
	let count = units;
	for (let i = 0; i < units - 1; i++) if ((str.charCodeAt(i) & 64512) === 55296 && (str.charCodeAt(i + 1) & 64512) === 56320) {
		count--;
		i++;
	}
	return count;
}
function getLengthableOrigin(input) {
	if (Array.isArray(input)) return "array";
	if (typeof input === "string") return "string";
	return "unknown";
}
function issue(...args) {
	const [iss, input, inst] = args;
	if (typeof iss === "string") return {
		message: iss,
		code: "custom",
		input,
		inst
	};
	return { ...iss };
}
/**
* Installs a trait's members on its prototype. Each value builds that member for the instance on first read; the built value shadows the accessor as an own property, so a detached `const { parse } = schema` keeps working.
*
* Call this from a `proto` initializer, which runs once per prototype — never per instance.
*/
function members(proto, table) {
	for (const key in table) {
		const desc = Object.getOwnPropertyDescriptor(table, key);
		if (desc.get) Object.defineProperty(proto, key, {
			...desc,
			enumerable: false
		});
		else defineBound(proto, key, desc.value);
	}
}
/** Shadows a prototype member with an own value, so a getter that builds from the instance runs once. */
function own(inst, key, value, enumerable = true) {
	Object.defineProperty(inst, key, {
		configurable: true,
		writable: true,
		enumerable,
		value
	});
	return value;
}
/** Like {@link own}, for a member that was never an own data property and has to stay out of `Object.keys`. */
function hide(inst, key, value) {
	return own(inst, key, value, false);
}
function defineBound(proto, key, fn) {
	Object.defineProperty(proto, key, {
		configurable: true,
		get() {
			return this == null ? fn : own(this, key, fn.bind(this));
		},
		set(value) {
			own(this, key, value);
		}
	});
}
/** Returns the prototype to install on, or `undefined` if this group is already installed on it. */
function claim(inst, sentinel) {
	const proto = Object.getPrototypeOf(inst);
	return sentinel in proto ? void 0 : proto;
}
var installing;
var broke = false;
var breaker = {
	configurable: true,
	get() {
		broke = true;
	}
};
/**
* Installs a lazily-derived internal on the `_zod` prototype of `inst`'s
* constructor, computed from the internals object itself and cached there on
* first read. One accessor per constructor rather than one per instance.
*/
function defineLazyInternal(inst, key, compute) {
	const proto = Object.getPrototypeOf(inst._zod);
	if (key in proto && installing !== inst._zod) {
		installing = void 0;
		return;
	}
	installing = inst._zod;
	Object.defineProperty(proto, key, {
		configurable: true,
		get() {
			Object.defineProperty(this, key, breaker);
			const outer = broke;
			broke = false;
			try {
				const value = compute(this);
				if (broke) delete this[key];
				else Object.defineProperty(this, key, {
					configurable: true,
					writable: true,
					value
				});
				broke = broke || outer;
				return value;
			} catch (err) {
				delete this[key];
				broke = broke || outer;
				throw err;
			}
		},
		set(value) {
			Object.defineProperty(this, key, {
				configurable: true,
				writable: true,
				value
			});
		}
	});
}
/**
* Installs `key` on `inst`'s prototype, computed by `make` on first read and cached there as an own
* data property. One accessor per constructor rather than one per instance, because an own accessor
* puts every instance after the first into v8 dictionary mode. The key doubles as the sentinel.
*/
function installLazyProp(inst, key, make, enumerable) {
	const proto = claim(inst, key);
	if (!proto) return;
	Object.defineProperty(proto, key, {
		configurable: true,
		get() {
			const desc = {
				configurable: true,
				writable: true,
				enumerable,
				value: void 0
			};
			Object.defineProperty(this, key, desc);
			desc.value = make(this);
			Object.defineProperty(this, key, desc);
			return desc.value;
		},
		set(value) {
			Object.defineProperty(this, key, {
				configurable: true,
				writable: true,
				enumerable,
				value
			});
		}
	});
}
// ../../node_modules/zod/v4/core/core.js
var _a;
var _zodDesc = {
	value: void 0,
	enumerable: false
};
var _E = "captureStackTrace" in Error ? Error : null;
function newError(Definition) {
	const E = _E;
	if (E) {
		const saved = E.stackTraceLimit;
		if (typeof saved === "number") {
			try {
				E.stackTraceLimit = 0;
			} catch {
				_E = null;
				return new Definition();
			}
			try {
				return new Definition();
			} finally {
				E.stackTraceLimit = saved;
			}
		}
	}
	return new Definition();
}
/* @__NO_SIDE_EFFECTS__ */
function $constructor(name, initializer, proto, params) {
	const zodProto = {};
	function Internals(def) {
		this.def = def;
		this.constr = _;
		this.traits = /* @__PURE__ */ new Set();
	}
	Internals.prototype = zodProto;
	const protoMembers = proto;
	const initialized = protoMembers && /* @__PURE__ */ new WeakSet();
	function init(inst, def) {
		if (!inst._zod) {
			_zodDesc.value = new Internals(def);
			try {
				Object.defineProperty(inst, "_zod", _zodDesc);
			} finally {
				_zodDesc.value = void 0;
			}
		} else if (inst._zod.traits.has(name)) return;
		inst._zod.traits.add(name);
		initializer(inst, def);
		if (initialized) {
			const own = Object.getPrototypeOf(inst);
			const ctorProto = inst._zod.constr.prototype;
			let up = own;
			while (up && up !== ctorProto) up = Object.getPrototypeOf(up);
			const target = up ?? own;
			if (!initialized.has(target)) {
				initialized.add(target);
				members(target, protoMembers);
			}
		}
		const proto = _.prototype;
		for (const k in proto) {
			if (!Object.prototype.hasOwnProperty.call(proto, k)) continue;
			if (!(k in inst)) inst[k] = proto[k].bind(inst);
		}
	}
	const Parent = params?.Parent ?? Object;
	class Definition extends Parent {}
	Object.defineProperty(Definition, "name", { value: name });
	function _(def) {
		const inst = params?.Parent ? newError(Definition) : this;
		init(inst, def);
		const deferred = inst._zod.deferred;
		if (deferred) {
			for (const fn of deferred) fn();
			inst._zod.deferred = void 0;
		}
		const pp = globalThis.__zod_globalConfig?.postProcessor;
		if (pp) pp(inst);
		return inst;
	}
	Object.defineProperty(_, "init", { value: init });
	Object.defineProperty(_, Symbol.hasInstance, { value: (inst) => {
		if (params?.Parent && inst instanceof params.Parent) return true;
		return inst?._zod?.traits?.has(name);
	} });
	Object.defineProperty(_, "name", { value: name });
	return _;
}
var $ZodAsyncError = class extends Error {
	constructor() {
		super(`Encountered Promise during synchronous parse. Use .parseAsync() instead.`);
	}
};
(_a = globalThis).__zod_globalConfig ?? (_a.__zod_globalConfig = {});
var globalConfig = globalThis.__zod_globalConfig;
function config(newConfig) {
	if (newConfig) Object.assign(globalConfig, newConfig);
	return globalConfig;
}
// ../../node_modules/zod/v4/core/errors.js
function _getMessage() {
	const internals = this._zod;
	internals.message ?? (internals.message = JSON.stringify(internals.def, jsonStringifyReplacer, 2));
	return internals.message;
}
function _setMessage(value) {
	this._zod.message = value;
}
var _messageDesc = {
	get: _getMessage,
	set: _setMessage,
	enumerable: true,
	configurable: true
};
var _issuesDesc = {
	value: void 0,
	enumerable: false
};
var _installedToString = /* @__PURE__ */ new WeakSet([Object.prototype, Error.prototype]);
var initializer = (inst, def) => {
	inst.name = "$ZodError";
	_issuesDesc.value = def;
	Object.defineProperty(inst, "issues", _issuesDesc);
	_issuesDesc.value = void 0;
	Object.defineProperty(inst, "message", _messageDesc);
	const proto = Object.getPrototypeOf(inst);
	if (!_installedToString.has(proto)) {
		_installedToString.add(proto);
		Object.defineProperty(proto, "toString", {
			configurable: true,
			enumerable: false,
			get() {
				const value = () => this.message;
				Object.defineProperty(this, "toString", {
					value,
					configurable: true,
					writable: true
				});
				return value;
			},
			set(value) {
				Object.defineProperty(this, "toString", {
					value,
					configurable: true,
					writable: true
				});
			}
		});
	}
};
var $ZodRealError = /* @__PURE__ */ $constructor("$ZodError", initializer, void 0, { Parent: Error });
// ../../node_modules/zod/v4/core/parse.js
var _parse = (_Err) => {
	const fn = (schema, value, _ctx, _params) => {
		const ctx = _ctx ? {
			..._ctx,
			async: false
		} : { async: false };
		const result = schema._zod.run({
			value,
			issues: []
		}, ctx);
		if (result instanceof Promise) throw new $ZodAsyncError();
		if (result.issues.length) {
			const e = new ((_params?.Err) ?? _Err)(result.issues.map((iss) => finalizeIssue(iss, ctx, config())));
			captureStackTrace(e, _params?.callee ?? fn);
			throw e;
		}
		return result.value;
	};
	return fn;
};
var parse = /* @__PURE__*/ _parse($ZodRealError);
var _parseAsync = (_Err) => {
	const fn = async (schema, value, _ctx, params) => {
		const ctx = _ctx ? {
			..._ctx,
			async: true
		} : { async: true };
		let result = schema._zod.run({
			value,
			issues: []
		}, ctx);
		if (result instanceof Promise) result = await result;
		if (result.issues.length) {
			const e = new ((params?.Err) ?? _Err)(result.issues.map((iss) => finalizeIssue(iss, ctx, config())));
			captureStackTrace(e, params?.callee ?? fn);
			throw e;
		}
		return result.value;
	};
	return fn;
};
var parseAsync = /* @__PURE__*/ _parseAsync($ZodRealError);
var _safeParse = (_Err) => (schema, value, _ctx) => {
	const ctx = _ctx ? {
		..._ctx,
		async: false
	} : { async: false };
	const result = schema._zod.run({
		value,
		issues: []
	}, ctx);
	if (result instanceof Promise) throw new $ZodAsyncError();
	return result.issues.length ? failure(_Err, result.issues, ctx) : {
		success: true,
		data: result.value
	};
};
var safeParse = /* @__PURE__*/ _safeParse($ZodRealError);
function failure(Err, issues, ctx) {
	let error;
	return {
		success: false,
		get error() {
			if (!error) {
				error = new Err(issues.map((iss) => finalizeIssue(iss, ctx, config())));
				issues = void 0;
				ctx = void 0;
			}
			return error;
		},
		set error(e) {
			error = e;
			issues = void 0;
			ctx = void 0;
		}
	};
}
var _safeParseAsync = (_Err) => async (schema, value, _ctx) => {
	const ctx = _ctx ? {
		..._ctx,
		async: true
	} : { async: true };
	let result = schema._zod.run({
		value,
		issues: []
	}, ctx);
	if (result instanceof Promise) result = await result;
	return result.issues.length ? failure(_Err, result.issues, ctx) : {
		success: true,
		data: result.value
	};
};
var safeParseAsync = /* @__PURE__*/ _safeParseAsync($ZodRealError);
// ../../node_modules/zod/v4/core/regexes.js
var anyString = /^[\s\S]{0,}$/;
var number$1 = /^-?\d+(?:\.\d+)?$/;
// ../../node_modules/zod/v4/core/checks.js
var $ZodCheck = /*@__PURE__*/ $constructor("$ZodCheck", (inst, def) => {
	var _a;
	inst._zod ?? (inst._zod = {});
	inst._zod.def = def;
	(_a = inst._zod).onattach ?? (_a.onattach = []);
});
/** Default `when` for length-based checks: run only on non-nullish values with a `length`. */
var _whenHasLength = (payload) => {
	const val = payload.value;
	return !nullish(val) && val.length !== void 0;
};
var numericOriginMap = {
	number: "number",
	bigint: "bigint",
	object: "date"
};
var $ZodCheckLessThan = /*@__PURE__*/ $constructor("$ZodCheckLessThan", (inst, def) => {
	$ZodCheck.init(inst, def);
	const origin = numericOriginMap[typeof def.value];
	inst._zod.check = (payload) => {
		if (def.inclusive ? payload.value <= def.value : payload.value < def.value) return;
		payload.issues.push({
			origin: numericOriginMap[typeof payload.value] ?? origin,
			code: "too_big",
			maximum: typeof def.value === "object" ? def.value.getTime() : def.value,
			input: payload.value,
			inclusive: def.inclusive,
			inst,
			continue: !def.abort
		});
	};
});
var $ZodCheckGreaterThan = /*@__PURE__*/ $constructor("$ZodCheckGreaterThan", (inst, def) => {
	$ZodCheck.init(inst, def);
	const origin = numericOriginMap[typeof def.value];
	inst._zod.check = (payload) => {
		if (def.inclusive ? payload.value >= def.value : payload.value > def.value) return;
		payload.issues.push({
			origin: numericOriginMap[typeof payload.value] ?? origin,
			code: "too_small",
			minimum: typeof def.value === "object" ? def.value.getTime() : def.value,
			input: payload.value,
			inclusive: def.inclusive,
			inst,
			continue: !def.abort
		});
	};
});
var $ZodCheckNumberFormat = /*@__PURE__*/ $constructor("$ZodCheckNumberFormat", (inst, def) => {
	$ZodCheck.init(inst, def);
	def.format = def.format || "float64";
	const isInt = def.format?.includes("int");
	const origin = isInt ? "int" : "number";
	const [minimum, maximum] = NUMBER_FORMAT_RANGES[def.format];
	inst._zod.check = (payload) => {
		const input = payload.value;
		if (isInt) {
			if (!Number.isInteger(input)) {
				payload.issues.push({
					expected: origin,
					format: def.format,
					code: "invalid_type",
					continue: false,
					input,
					inst
				});
				return;
			}
			if (!Number.isSafeInteger(input)) {
				if (input > 0) payload.issues.push({
					input,
					code: "too_big",
					maximum: Number.MAX_SAFE_INTEGER,
					note: "Integers must be within the safe integer range.",
					inst,
					origin,
					inclusive: true,
					continue: !def.abort
				});
				else payload.issues.push({
					input,
					code: "too_small",
					minimum: Number.MIN_SAFE_INTEGER,
					note: "Integers must be within the safe integer range.",
					inst,
					origin,
					inclusive: true,
					continue: !def.abort
				});
				return;
			}
		}
		if (input < minimum) payload.issues.push({
			origin: "number",
			input,
			code: "too_small",
			minimum,
			inclusive: true,
			inst,
			continue: !def.abort
		});
		if (input > maximum) payload.issues.push({
			origin: "number",
			input,
			code: "too_big",
			maximum,
			inclusive: true,
			inst,
			continue: !def.abort
		});
	};
});
var $ZodCheckMaxLength = /*@__PURE__*/ $constructor("$ZodCheckMaxLength", (inst, def) => {
	var _a;
	$ZodCheck.init(inst, def);
	(_a = inst._zod.def).when ?? (_a.when = _whenHasLength);
	inst._zod.check = (payload) => {
		const input = payload.value;
		const units = input.length;
		if ((typeof input === "string" && units > def.maximum ? codePointLength(input) : units) <= def.maximum) return;
		const origin = getLengthableOrigin(input);
		payload.issues.push({
			origin,
			code: "too_big",
			maximum: def.maximum,
			inclusive: true,
			input,
			inst,
			continue: !def.abort
		});
	};
});
var $ZodCheckMinLength = /*@__PURE__*/ $constructor("$ZodCheckMinLength", (inst, def) => {
	var _a;
	$ZodCheck.init(inst, def);
	(_a = inst._zod.def).when ?? (_a.when = _whenHasLength);
	inst._zod.check = (payload) => {
		const input = payload.value;
		const units = input.length;
		if ((typeof input === "string" && units >= def.minimum && units < def.minimum * 2 ? codePointLength(input) : units) >= def.minimum) return;
		const origin = getLengthableOrigin(input);
		payload.issues.push({
			origin,
			code: "too_small",
			minimum: def.minimum,
			inclusive: true,
			input,
			inst,
			continue: !def.abort
		});
	};
});
var $ZodCheckStringFormat = /*@__PURE__*/ $constructor("$ZodCheckStringFormat", (inst, def) => {
	var _a, _b;
	$ZodCheck.init(inst, def);
	if (def.pattern) (_a = inst._zod).check ?? (_a.check = (payload) => {
		def.pattern.lastIndex = 0;
		if (def.pattern.test(payload.value)) return;
		payload.issues.push({
			origin: "string",
			code: "invalid_format",
			format: def.format,
			input: payload.value,
			...def.pattern ? { pattern: def.pattern.toString() } : {},
			inst,
			continue: !def.abort
		});
	});
	else (_b = inst._zod).check ?? (_b.check = () => {});
});
var $ZodCheckRegex = /*@__PURE__*/ $constructor("$ZodCheckRegex", (inst, def) => {
	$ZodCheckStringFormat.init(inst, def);
	inst._zod.check = (payload) => {
		def.pattern.lastIndex = 0;
		if (def.pattern.test(payload.value)) return;
		payload.issues.push({
			origin: "string",
			code: "invalid_format",
			format: "regex",
			input: payload.value,
			pattern: def.pattern.toString(),
			inst,
			continue: !def.abort
		});
	};
});
// ../../node_modules/zod/v4/core/versions.js
var version = {
	major: 4,
	minor: 6,
	patch: 7
};
// ../../node_modules/zod/v4/core/schemas.js
var $ZodType = /*@__PURE__*/ $constructor("$ZodType", (inst, def) => {
	var _a;
	inst ?? (inst = {});
	inst._zod.def = def;
	inst._zod.bag = inst._zod.bag || {};
	inst._zod.version = version;
	const defChecks = inst._zod.def.checks;
	const checks = inst._zod.traits.has("$ZodCheck") ? [inst, ...defChecks ?? []] : defChecks?.length ? [...defChecks] : [];
	for (const ch of checks) for (const fn of ch._zod.onattach) fn(inst);
	if (checks.length === 0) {
		(_a = inst._zod).deferred ?? (_a.deferred = []);
		inst._zod.deferred?.push(() => {
			inst._zod.run = inst._zod.parse;
		});
	} else {
		const runChecks = (payload, checks, ctx) => {
			if (payload.memo) return payload;
			let isAborted = aborted(payload);
			let asyncResult;
			for (const ch of checks) {
				if (ch._zod.def.when) {
					if (explicitlyAborted(payload)) continue;
					if (!ch._zod.def.when(payload)) continue;
				} else if (isAborted) continue;
				const currLen = payload.issues.length;
				const _ = ch._zod.check(payload);
				if (_ instanceof Promise && ctx?.async === false) throw new $ZodAsyncError();
				if (asyncResult || _ instanceof Promise) asyncResult = (asyncResult ?? Promise.resolve()).then(async () => {
					await _;
					if (payload.issues.length === currLen) return;
					attachSchema(payload.issues, currLen, inst);
					if (!isAborted) isAborted = aborted(payload, currLen);
				});
				else {
					if (payload.issues.length === currLen) continue;
					attachSchema(payload.issues, currLen, inst);
					if (!isAborted) isAborted = aborted(payload, currLen);
				}
			}
			if (asyncResult) return asyncResult.then(() => {
				return payload;
			});
			return payload;
		};
		const handleCanaryResult = (canary, payload, ctx) => {
			if (aborted(canary)) {
				canary.aborted = true;
				return canary;
			}
			const checkResult = runChecks(payload, checks, ctx);
			if (checkResult instanceof Promise) {
				if (ctx.async === false) throw new $ZodAsyncError();
				return checkResult.then((checkResult) => inst._zod.parse(checkResult, ctx));
			}
			return inst._zod.parse(checkResult, ctx);
		};
		inst._zod.run = (payload, ctx) => {
			if (ctx.skipChecks) return inst._zod.parse(payload, ctx);
			if (ctx.direction === "backward") {
				const canary = inst._zod.parse({
					value: payload.value,
					issues: []
				}, {
					...ctx,
					skipChecks: true
				});
				if (canary instanceof Promise) return canary.then((canary) => {
					return handleCanaryResult(canary, payload, ctx);
				});
				return handleCanaryResult(canary, payload, ctx);
			}
			const result = inst._zod.parse(payload, ctx);
			if (result instanceof Promise) {
				if (ctx.async === false) throw new $ZodAsyncError();
				return result.then((result) => runChecks(result, checks, ctx));
			}
			return runChecks(result, checks, ctx);
		};
	}
}, {
	get "~standard"() {
		return hide(this, "~standard", standardProps(this));
	},
	set "~standard"(value) {
		own(this, "~standard", value);
	}
});
/** The Standard Schema surface for `inst`. Shared so wrappers can extend it without forcing it. */
var toStandardResult = (r, ctx) => r.issues.length ? { issues: r.issues.map((iss) => finalizeIssue(iss, ctx, config())) } : { value: r.value };
async function validateAsync(inst, value) {
	const ctx = { async: true };
	return toStandardResult(await inst._zod.run({
		value,
		issues: []
	}, ctx), ctx);
}
function standardProps(inst) {
	return {
		validate: (value) => {
			const ctx = { async: false };
			try {
				const r = inst._zod.run({
					value,
					issues: []
				}, ctx);
				if (!(r instanceof Promise)) return toStandardResult(r, ctx);
			} catch (_) {}
			return validateAsync(inst, value);
		},
		vendor: "zod",
		version: 1
	};
}
var $ZodString = /*@__PURE__*/ $constructor("$ZodString", (inst, def) => {
	$ZodType.init(inst, def);
	inst._zod.pattern = def.pattern ?? anyString;
	inst._zod.parse = (payload, _) => {
		if (def.coerce) try {
			payload.value = String(payload.value);
		} catch (_) {}
		if (typeof payload.value === "string") return payload;
		payload.issues.push({
			expected: "string",
			code: "invalid_type",
			input: payload.value,
			inst
		});
		return payload;
	};
});
var $ZodNumber = /*@__PURE__*/ $constructor("$ZodNumber", (inst, def) => {
	$ZodType.init(inst, def);
	inst._zod.pattern = number$1;
	inst._zod.parse = (payload, _ctx) => {
		if (def.coerce) try {
			payload.value = Number(payload.value);
		} catch (_) {}
		const input = payload.value;
		if (typeof input === "number" && !Number.isNaN(input) && Number.isFinite(input)) return payload;
		const received = typeof input === "number" ? Number.isNaN(input) ? "NaN" : !Number.isFinite(input) ? String(input) : void 0 : void 0;
		payload.issues.push({
			expected: "number",
			code: "invalid_type",
			input,
			inst,
			...received ? { received } : {}
		});
		return payload;
	};
});
var $ZodNumberFormat = /*@__PURE__*/ $constructor("$ZodNumberFormat", (inst, def) => {
	$ZodCheckNumberFormat.init(inst, def);
	$ZodNumber.init(inst, def);
});
function capSize(inst, size) {
	const caps = sizeCaps(inst._zod.def.checks);
	if (!caps.length) return;
	const cap = Math.min(...caps.map((c) => c[0]));
	const parse = inst._zod.parse;
	inst._zod.parse = (payload, ctx) => {
		const n = size(payload.value);
		if (n > cap && !ctx?.skipChecks && ctx?.direction !== "backward") {
			for (const [max, ch] of sizeCaps(inst._zod.def.checks)) if (n > max && ch._zod.def.when(payload)) {
				const start = payload.issues.length;
				ch._zod.check(payload);
				payload.issues[start].continue = false;
				attachSchema(payload.issues, start, inst);
				return payload;
			}
		}
		return parse(payload, ctx);
	};
}
var arraySize = (input) => Array.isArray(input) ? input.length : -1;
function handleArrayResult(result, final, index) {
	if (result.issues.length) final.issues.push(...prefixIssues(index, result.issues));
	final.value[index] = result.value;
}
var $ZodArray = /*@__PURE__*/ $constructor("$ZodArray", (inst, def) => {
	$ZodType.init(inst, def);
	const memo = globalConfig.memoizer;
	memo?.attach(inst);
	inst._zod.parse = (payload, ctx) => {
		const input = payload.value;
		if (!Array.isArray(input)) {
			payload.issues.push({
				expected: "array",
				code: "invalid_type",
				input,
				inst
			});
			return payload;
		}
		payload.value = memo ? memo.alloc(inst, payload, Array(input.length), ctx) : Array(input.length);
		const proms = [];
		const abortEarly = ctx?.abortEarly;
		for (let i = 0; i < input.length; i++) {
			const item = input[i];
			const result = def.element._zod.run({
				value: item,
				issues: []
			}, ctx);
			if (result instanceof Promise) proms.push(result.then((result) => handleArrayResult(result, payload, i)));
			else {
				handleArrayResult(result, payload, i);
				if (abortEarly && result.issues.length !== 0 && aborted(result)) break;
			}
		}
		if (proms.length) return Promise.all(proms).then(() => payload);
		return payload;
	};
	capSize(inst, arraySize);
});
function handlePropertyResult(result, final, key, input, optin, optout) {
	const isPresent = key in input;
	const isOptionalOut = optout === "optional";
	if (!isPresent && isOptionalOut && optin === "optional") return;
	if (result.issues.length) {
		if (optin !== void 0 && isOptionalOut && !isPresent) return;
		final.issues.push(...prefixIssues(key, result.issues));
	}
	if (!isPresent && optin === void 0) {
		if (!result.issues.length) final.issues.push({
			code: "invalid_type",
			expected: "nonoptional",
			input: void 0,
			path: [key]
		});
		return;
	}
	if (result.value === void 0) {
		if (isPresent || optin === "defaulted" && !isOptionalOut) final.value[key] = void 0;
	} else final.value[key] = result.value;
}
var NO_SYMBOL_KEYS = [];
function normalizeDef(def) {
	const keys = Object.keys(def.shape);
	const ownSymbols = Object.getOwnPropertySymbols(def.shape);
	const symbolKeys = ownSymbols.length ? ownSymbols : NO_SYMBOL_KEYS;
	const allKeys = symbolKeys.length ? [...keys, ...symbolKeys] : keys;
	for (const k of allKeys) if (!def.shape?.[k]?._zod?.traits?.has("$ZodType")) throw new Error(`Invalid element at key "${String(k)}": expected a Zod schema`);
	const okeys = optionalKeys(def.shape);
	return {
		...def,
		allKeys,
		symbolKeys,
		keySet: new Set(keys),
		numKeys: keys.length,
		optionalKeys: new Set(okeys)
	};
}
function handleCatchall(proms, input, payload, ctx, def, inst, abortEarly) {
	const unrecognized = [];
	const keySet = def.keySet;
	const _catchall = def.catchall._zod;
	const t = _catchall.def.type;
	const optin = _catchall.optin;
	const optout = _catchall.optout;
	let seen = 0;
	for (const key in input) {
		if (abortEarly && payload.issues.length !== seen) {
			if (aborted(payload, seen)) break;
			seen = payload.issues.length;
		}
		if (keySet.has(key)) continue;
		if (key === "__proto__") {
			if (t === "never") unrecognized.push(key);
			continue;
		}
		if (t === "never") {
			unrecognized.push(key);
			continue;
		}
		const r = _catchall.run({
			value: input[key],
			issues: []
		}, ctx);
		if (r instanceof Promise) proms.push(r.then((r) => handlePropertyResult(r, payload, key, input, optin, optout)));
		else handlePropertyResult(r, payload, key, input, optin, optout);
	}
	if (unrecognized.length) payload.issues.push({
		code: "unrecognized_keys",
		keys: unrecognized,
		input,
		inst,
		continue: true
	});
	if (!proms.length) return payload;
	return Promise.all(proms).then(() => {
		return payload;
	});
}
var $ZodObject = /*@__PURE__*/ $constructor("$ZodObject", (inst, def) => {
	$ZodType.init(inst, def);
	const desc = Object.getOwnPropertyDescriptor(def, "shape");
	const sh = desc?.get ? desc.get.raw : def.shape ?? {};
	if (sh) {
		const get = () => {
			const newSh = { ...sh };
			Object.defineProperty(def, "shape", { value: newSh });
			get.raw = newSh;
			return newSh;
		};
		get.raw = sh;
		Object.defineProperty(def, "shape", { get });
	}
	const _normalized = cached(() => normalizeDef(def));
	defineLazyInternal(inst, "propValues", (zod) => {
		const shape = zod.def.shape;
		const propValues = {};
		for (const key in shape) {
			const field = shape[key]._zod;
			if (field.values) {
				if (!Object.prototype.hasOwnProperty.call(propValues, key)) assignProp(propValues, key, /* @__PURE__ */ new Set());
				for (const v of field.values) propValues[key].add(v);
				if (field.optin !== void 0) propValues[key].add(void 0);
			}
		}
		return propValues;
	});
	const isObject$1 = isObject;
	const catchall = def.catchall;
	let value;
	const memo = globalConfig.memoizer;
	memo?.attach(inst);
	inst._zod.parse = (payload, ctx) => {
		value ?? (value = _normalized.value);
		const input = payload.value;
		if (!isObject$1(input)) {
			payload.issues.push({
				expected: "object",
				code: "invalid_type",
				input,
				inst
			});
			return payload;
		}
		payload.value = memo ? memo.alloc(inst, payload, {}, ctx) : {};
		const proms = [];
		const shape = value.shape;
		const abortEarly = ctx?.abortEarly;
		let seen = payload.issues.length;
		for (const key of value.allKeys) {
			if (abortEarly && payload.issues.length !== seen) {
				if (aborted(payload, seen)) break;
				seen = payload.issues.length;
			}
			if (key === "__proto__") continue;
			const el = shape[key];
			const optin = el._zod.optin;
			const optout = el._zod.optout;
			const r = el._zod.run({
				value: input[key],
				issues: []
			}, ctx);
			if (r instanceof Promise) proms.push(r.then((r) => handlePropertyResult(r, payload, key, input, optin, optout)));
			else handlePropertyResult(r, payload, key, input, optin, optout);
		}
		if (!catchall) return proms.length ? Promise.all(proms).then(() => payload) : payload;
		return handleCatchall(proms, input, payload, ctx, _normalized.value, inst, abortEarly === true);
	};
});
var $ZodEnum = /*@__PURE__*/ $constructor("$ZodEnum", (inst, def) => {
	$ZodType.init(inst, def);
	const values = getEnumValues(def.entries);
	const valuesSet = new Set(values);
	inst._zod.values = valuesSet;
	defineLazyInternal(inst, "pattern", (zod) => {
		const patternValues = getEnumValues(zod.def.entries).filter((k) => propertyKeyTypes.has(typeof k));
		return new RegExp(patternValues.length ? `^(${patternValues.map((o) => escapeRegex(o.toString())).join("|")})$` : "^[^\\s\\S]$");
	});
	inst._zod.parse = (payload, _ctx) => {
		const input = payload.value;
		if (valuesSet.has(input)) return payload;
		payload.issues.push({
			code: "invalid_value",
			values,
			input,
			inst
		});
		return payload;
	};
});
function handleOptionalResult(payload, result) {
	payload.value = result.issues.length ? void 0 : result.value;
	return payload;
}
var $ZodOptional = /*@__PURE__*/ $constructor("$ZodOptional", (inst, def) => {
	$ZodType.init(inst, def);
	defineLazyInternal(inst, "optin", (zod) => zod.def.innerType._zod.optin === "defaulted" ? "defaulted" : "optional");
	inst._zod.optout = "optional";
	defineLazyInternal(inst, "values", (zod) => {
		const values = zod.def.innerType._zod.values;
		return values ? /* @__PURE__ */ new Set([...values, void 0]) : void 0;
	});
	defineLazyInternal(inst, "pattern", (zod) => {
		const pattern = zod.def.innerType._zod.pattern;
		return pattern ? new RegExp(`^(${cleanRegex(pattern.source)})?$`) : void 0;
	});
	inst._zod.parse = (payload, ctx) => {
		if (payload.value === void 0) {
			if (def.innerType._zod.optin !== "defaulted") return payload;
			const result = def.innerType._zod.run({
				value: payload.value,
				issues: []
			}, ctx);
			if (result instanceof Promise) return result.then((result) => handleOptionalResult(payload, result));
			return handleOptionalResult(payload, result);
		}
		return def.innerType._zod.run(payload, ctx);
	};
});
var $ZodDefault = /*@__PURE__*/ $constructor("$ZodDefault", (inst, def) => {
	$ZodType.init(inst, def);
	inst._zod.optin = "defaulted";
	defineLazyInternal(inst, "values", (zod) => zod.def.innerType._zod.values);
	inst._zod.parse = (payload, ctx) => {
		if (ctx.direction === "backward") return def.innerType._zod.run(payload, ctx);
		if (payload.value === void 0) {
			payload.value = def.defaultValue;
			/**
			* $ZodDefault returns the default value immediately in forward direction.
			* It doesn't pass the default value into the validator ("prefault"). There's no reason to pass the default value through validation. The validity of the default is enforced by TypeScript statically. Otherwise, it's the responsibility of the user to ensure the default is valid. In the case of pipes with divergent in/out types, you can specify the default on the `in` schema of your ZodPipe to set a "prefault" for the pipe.   */
			return payload;
		}
		const result = def.innerType._zod.run(payload, ctx);
		if (result instanceof Promise) return result.then((result) => handleDefaultResult(result, def));
		return handleDefaultResult(result, def);
	};
});
function handleDefaultResult(payload, def) {
	if (payload.value === void 0) payload.value = def.defaultValue;
	return payload;
}
var $ZodCustom = /*@__PURE__*/ $constructor("$ZodCustom", (inst, def) => {
	$ZodCheck.init(inst, def);
	$ZodType.init(inst, def);
	inst._zod.parse = (payload, _) => {
		return payload;
	};
	inst._zod.check = (payload) => {
		const input = payload.value;
		const r = def.fn(input);
		if (r instanceof Promise) return r.then((r) => handleRefineResult(r, payload, input, inst));
		handleRefineResult(r, payload, input, inst);
	};
});
function handleRefineResult(result, payload, input, inst) {
	if (!result) {
		const _iss = {
			code: "custom",
			input,
			inst,
			path: [...inst._zod.def.path ?? []],
			continue: !inst._zod.def.abort
		};
		if (inst._zod.def.params) _iss.params = inst._zod.def.params;
		payload.issues.push(issue(_iss));
	}
}
// ../../node_modules/zod/v4/core/api.js
function snapshotChecks(def) {
	if (def.checks) def.checks = [...def.checks];
	return def;
}
// @__NO_SIDE_EFFECTS__
function _string(Class, params) {
	return new Class(snapshotChecks({
		type: "string",
		...normalizeParams(params)
	}));
}
// @__NO_SIDE_EFFECTS__
function _number(Class, params) {
	return new Class(snapshotChecks({
		type: "number",
		checks: [],
		...normalizeParams(params)
	}));
}
// @__NO_SIDE_EFFECTS__
function _int(Class, params) {
	return new Class({
		type: "number",
		check: "number_format",
		abort: false,
		format: "safeint",
		...normalizeParams(params)
	});
}
// @__NO_SIDE_EFFECTS__
function _lte(value, params) {
	return new $ZodCheckLessThan({
		check: "less_than",
		...normalizeParams(params),
		value,
		inclusive: true
	});
}
// @__NO_SIDE_EFFECTS__
function _gt(value, params) {
	return new $ZodCheckGreaterThan({
		check: "greater_than",
		...normalizeParams(params),
		value,
		inclusive: false
	});
}
// @__NO_SIDE_EFFECTS__
function _gte(value, params) {
	return new $ZodCheckGreaterThan({
		check: "greater_than",
		...normalizeParams(params),
		value,
		inclusive: true
	});
}
// @__NO_SIDE_EFFECTS__
function _positive(params) {
	return /* @__PURE__ */ _gt(0, params);
}
// @__NO_SIDE_EFFECTS__
function _maxLength(maximum, params) {
	return new $ZodCheckMaxLength({
		check: "max_length",
		...normalizeParams(params),
		maximum
	});
}
// @__NO_SIDE_EFFECTS__
function _minLength(minimum, params) {
	return new $ZodCheckMinLength({
		check: "min_length",
		...normalizeParams(params),
		minimum
	});
}
// @__NO_SIDE_EFFECTS__
function _regex(pattern, params) {
	return new $ZodCheckRegex({
		check: "string_format",
		format: "regex",
		...normalizeParams(params),
		pattern
	});
}
// @__NO_SIDE_EFFECTS__
function _refine(Class, fn, _params) {
	return new Class({
		type: "custom",
		check: "custom",
		fn,
		...normalizeParams(_params)
	});
}
// ../../node_modules/zod/v4/mini/schemas.js
var ZodMiniType = /*@__PURE__*/ $constructor("ZodMiniType", (inst, def) => {
	if (!inst._zod) throw new Error("Uninitialized schema in ZodMiniType.");
	$ZodType.init(inst, def);
	inst.def = def;
	inst.type = def.type;
}, {
	get with() {
		return this.check;
	},
	set with(value) {
		own(this, "with", value);
	},
	parse(data, params) {
		return parse(this, data, params, { callee: this.parse });
	},
	parseAsync(data, params) {
		return parseAsync(this, data, params, { callee: this.parseAsync });
	},
	safeParse(data, params) {
		return safeParse(this, data, params);
	},
	safeParseAsync(data, params) {
		return safeParseAsync(this, data, params);
	},
	check(...checks) {
		const def = this.def;
		return this.clone({
			...def,
			checks: [...def.checks ?? [], ...checks.map((ch) => typeof ch === "function" ? { _zod: {
				check: ch,
				def: { check: "custom" },
				onattach: []
			} } : ch)]
		}, { parent: true });
	},
	clone(_def, params) {
		return clone(this, _def, params);
	},
	brand() {
		return this;
	},
	register(reg, meta) {
		reg.add(this, meta);
		return this;
	},
	apply(fn, ...args) {
		return args.length === 0 ? fn(this) : fn(this, ...args);
	}
});
var ZodMiniString = /*@__PURE__*/ $constructor("ZodMiniString", (inst, def) => {
	$ZodString.init(inst, def);
	ZodMiniType.init(inst, def);
});
// @__NO_SIDE_EFFECTS__
function string(params) {
	return /* @__PURE__ */ _string(ZodMiniString, params);
}
var ZodMiniNumber = /*@__PURE__*/ $constructor("ZodMiniNumber", (inst, def) => {
	$ZodNumber.init(inst, def);
	ZodMiniType.init(inst, def);
});
// @__NO_SIDE_EFFECTS__
function number(params) {
	return /* @__PURE__ */ _number(ZodMiniNumber, params);
}
var ZodMiniNumberFormat = /*@__PURE__*/ $constructor("ZodMiniNumberFormat", (inst, def) => {
	$ZodNumberFormat.init(inst, def);
	ZodMiniNumber.init(inst, def);
});
function int(params) {
	return /* @__PURE__ */ _int(ZodMiniNumberFormat, params);
}
var ZodMiniArray = /*@__PURE__*/ $constructor("ZodMiniArray", (inst, def) => {
	$ZodArray.init(inst, def);
	ZodMiniType.init(inst, def);
});
// @__NO_SIDE_EFFECTS__
function array(element, params) {
	return new ZodMiniArray({
		type: "array",
		element,
		...normalizeParams(params)
	});
}
var ZodMiniObject = /*@__PURE__*/ $constructor("ZodMiniObject", (inst, def) => {
	$ZodObject.init(inst, def);
	ZodMiniType.init(inst, def);
	installLazyProp(inst, "shape", (self) => self._zod.def.shape, false);
});
// @__NO_SIDE_EFFECTS__
function object(shape, params) {
	return new ZodMiniObject({
		type: "object",
		shape: shape ?? {},
		...normalizeParams(params)
	});
}
var ZodMiniEnum = /*@__PURE__*/ $constructor("ZodMiniEnum", (inst, def) => {
	$ZodEnum.init(inst, def);
	ZodMiniType.init(inst, def);
	inst.options = [...inst._zod.values];
});
// @__NO_SIDE_EFFECTS__
function _enum(values, params) {
	return new ZodMiniEnum({
		type: "enum",
		entries: Array.isArray(values) ? Object.fromEntries(values.map((v) => [v, v])) : values,
		...normalizeParams(params)
	});
}
var ZodMiniOptional = /*@__PURE__*/ $constructor("ZodMiniOptional", (inst, def) => {
	$ZodOptional.init(inst, def);
	ZodMiniType.init(inst, def);
});
// @__NO_SIDE_EFFECTS__
function optional(innerType) {
	return new ZodMiniOptional({
		type: "optional",
		innerType
	});
}
var ZodMiniDefault = /*@__PURE__*/ $constructor("ZodMiniDefault", (inst, def) => {
	$ZodDefault.init(inst, def);
	ZodMiniType.init(inst, def);
});
// @__NO_SIDE_EFFECTS__
function _default(innerType, defaultValue) {
	return new ZodMiniDefault({
		type: "default",
		innerType,
		get defaultValue() {
			return typeof defaultValue === "function" ? defaultValue() : shallowClone(defaultValue);
		}
	});
}
var ZodMiniCustom = /*@__PURE__*/ $constructor("ZodMiniCustom", (inst, def) => {
	$ZodCustom.init(inst, def);
	ZodMiniType.init(inst, def);
});
// @__NO_SIDE_EFFECTS__
function refine(fn, _params = {}) {
	return /* @__PURE__ */ _refine(ZodMiniCustom, fn, _params);
}
var id = (/* @__PURE__ */ string()).check(/* @__PURE__ */ _minLength(1), /* @__PURE__ */ _maxLength(128));
var shortText = (/* @__PURE__ */ string()).check(/* @__PURE__ */ _maxLength(64));
(/* @__PURE__ */ string()).check(/* @__PURE__ */ _maxLength(256), /* @__PURE__ */ _regex(/^https?:\/\/d-[a-z0-9]{16}\.[a-z0-9.-]+(:[0-9]{1,5})?$/));
(/* @__PURE__ */ string()).check(/* @__PURE__ */ _regex(/^pur_[A-Za-z0-9_-]{22}$/));
var range = (min, max) => (/* @__PURE__ */ number()).check(/* @__PURE__ */ _gte(min), /* @__PURE__ */ _lte(max));
var playerSummarySchema = /* @__PURE__ */ object({
	id,
	name: shortText
});
var voiceModeSchema = /* @__PURE__ */ _enum([
	"open",
	"host_only",
	"turn",
	"allowlist"
]);
/** 音高分析的模式：ptt＝只在有發言權且按住說話（或開麥）時分析（第一版只有這個） */
var pitchModeSchema = /* @__PURE__ */ _enum(["ptt"]);
/** 音高結果給誰看：self＝只送給唱歌者自己的遊戲畫面（第一版只有這個） */
var pitchAudienceSchema = /* @__PURE__ */ _enum(["self"]);
(/* @__PURE__ */ string()).check(/* @__PURE__ */ _maxLength(32)), range(0, 1), int().check(/* @__PURE__ */ _gte(0), /* @__PURE__ */ _lte(1e3)), range(0, 86400);
range(0, 1e4), range(0, 1e4), range(0, 1e4), range(0, 1e4);
(/* @__PURE__ */ string()).check(/* @__PURE__ */ _minLength(16), /* @__PURE__ */ _maxLength(256)), (/* @__PURE__ */ string()).check(/* @__PURE__ */ _maxLength(16)), (/* @__PURE__ */ array(playerSummarySchema)).check(/* @__PURE__ */ _maxLength(64)), range(0, 1), int().check(/* @__PURE__ */ _gte(0)), range(0, 0xe8d4a51000), range(0, 0x5af3107a4000), range(20, 5e3), range(0, 127), range(-50, 50), range(0, 1);
(/* @__PURE__ */ string()).check(/* @__PURE__ */ _maxLength(64)), (/* @__PURE__ */ string()).check(/* @__PURE__ */ _minLength(1), /* @__PURE__ */ _maxLength(64)), (/* @__PURE__ */ string()).check(/* @__PURE__ */ _maxLength(64)), (/* @__PURE__ */ string()).check(/* @__PURE__ */ _maxLength(128));
/**
* 發言規則（docs/voice-chat.md）。由遊戲伺服器簽章送給平台。
* - `turn`：剛好一位發言者
* - `allowlist`：至少一位
* - `open` / `host_only`：不帶名單
*/
var voicePolicySchema = (/* @__PURE__ */ object({
	roomId: id,
	gameInstanceId: id,
	sequence: int().check(/* @__PURE__ */ _gte(0)),
	mode: voiceModeSchema,
	allowedSpeakerIds: /* @__PURE__ */ _default((/* @__PURE__ */ array(id)).check(/* @__PURE__ */ _maxLength(64)), []),
	effectiveAtServerMs: int().check(/* @__PURE__ */ _positive()),
	expiresAtServerMs: int().check(/* @__PURE__ */ _positive())
})).check(/* @__PURE__ */ refine((p) => p.expiresAtServerMs > p.effectiveAtServerMs, "expiresAtServerMs 必須晚於 effectiveAtServerMs"), /* @__PURE__ */ refine((p) => p.mode !== "turn" || p.allowedSpeakerIds.length === 1, "turn 模式必須剛好一位發言者"), /* @__PURE__ */ refine((p) => p.mode !== "allowlist" || p.allowedSpeakerIds.length >= 1, "allowlist 模式至少要一位發言者"), /* @__PURE__ */ refine((p) => p.mode !== "open" && p.mode !== "host_only" || p.allowedSpeakerIds.length === 0, "open / host_only 模式不帶發言者名單"), /* @__PURE__ */ refine((p) => new Set(p.allowedSpeakerIds).size === p.allowedSpeakerIds.length, "發言者名單不得重複"));
/**
* 音高授權（docs/voice-chat.md「三之一」）。由遊戲伺服器簽章送給平台，和發言規則分開。
* epoch 必須是房間目前的局數；有效期間最多 1 小時；名單 1–64 人
*/
var pitchPolicySchema = (/* @__PURE__ */ object({
	roomId: id,
	gameInstanceId: id,
	epoch: int().check(/* @__PURE__ */ _gte(0)),
	sequence: int().check(/* @__PURE__ */ _gte(0)),
	allowedPlayerIds: (/* @__PURE__ */ array(id)).check(/* @__PURE__ */ _minLength(1), /* @__PURE__ */ _maxLength(64)),
	mode: pitchModeSchema,
	audience: pitchAudienceSchema,
	effectiveAtServerMs: int().check(/* @__PURE__ */ _positive()),
	expiresAtServerMs: int().check(/* @__PURE__ */ _positive())
})).check(/* @__PURE__ */ refine((p) => p.expiresAtServerMs > p.effectiveAtServerMs, "expiresAtServerMs 必須晚於 effectiveAtServerMs"), /* @__PURE__ */ refine((p) => new Set(p.allowedPlayerIds).size === p.allowedPlayerIds.length, "名單不得重複"));
(/* @__PURE__ */ string()).check(/* @__PURE__ */ _minLength(16), /* @__PURE__ */ _maxLength(256));
var launchExchangeResponseSchema = /* @__PURE__ */ object({
	gameId: id,
	playerId: id,
	name: shortText,
	roomId: id,
	/** 房間目前的局數：房主「重新開始」時加一。遊戲伺服器依局數開新局，並拒絕舊局的連線 */
	epoch: int().check(/* @__PURE__ */ _gte(0)),
	/** 平台房主的假名 ID：遊戲的房主以此為準（平台的踢人、重新開始也由這個人操作） */
	hostPlayerId: id,
	/** 平台點數：這位玩家在這款遊戲擁有的永久解鎖 SKU（點數開啟時才有） */
	entitlements: /* @__PURE__ */ optional((/* @__PURE__ */ array((/* @__PURE__ */ string()).check(/* @__PURE__ */ _maxLength(64)))).check(/* @__PURE__ */ _maxLength(500)))
});
// src/gameserver.ts
/** onAction 丟出這個錯誤時，玩家收到 400 {"error": code} */
var ActionError = class extends Error {
	code;
	constructor(code) {
		super(code);
		this.code = code;
		this.name = "ActionError";
	}
};
var OFFLINE_AFTER_MS = 5e3;
var IDLE_ROOM_MS = 18e5;
var STALE_SYNCS = 6;
var MAX_QUEUED_REPORTS = 1e4;
/** 回報一局失敗時重試前等待的時間 */
var retryDelaysMs = [
	1e3,
	4e3,
	15e3
];
var positive = (v, fallback) => typeof v === "number" && Number.isFinite(v) && v > 0 ? v : fallback;
function envDuration(name, fallbackMs, log) {
	const v = process.env[name];
	if (!v) return fallbackMs;
	const m = /^(\d+(?:\.\d+)?)(ms|s|m)$/.exec(v);
	const ms = m ? Number(m[1]) * {
		ms: 1,
		s: 1e3,
		m: 6e4
	}[m[2]] : 0;
	if (ms > 0) return ms;
	log(`${name} 格式錯誤（例如 10s、1m），改用預設 ${fallbackMs}ms`);
	return fallbackMs;
}
var newToken = () => randomBytes(16).toString("hex");
var AI_PATHS = {
	anthropic: "/anthropic",
	openai: "/openai/v1",
	gemini: "/gemini"
};
/** 指定服務要給官方套件的網址（anthropic、openai、gemini；OpenAI 的網址含 /v1） */
function aiBaseURL(provider) {
	const path = Object.hasOwn(AI_PATHS, provider) ? AI_PATHS[provider] : void 0;
	if (!path) throw new Error(`玩聚 AI 閘道沒有這個服務：${String(provider)}（可用 anthropic、openai、gemini）`);
	const base = (process.env.PT_AI_URL || "").replace(/\/+$/, "");
	if (!base) throw new Error("沒有 PT_AI_URL：這個部署沒有玩聚 AI 閘道（本機開發請用 pt dev）");
	return base + path;
}
/** 部署專用的 AI 代號（每次啟動時讀；重新部署會換新）。不要寫進紀錄 */
function aiToken() {
	const t = readFileSync(process.env.PT_AI_TOKEN_FILE || "/run/pt/ai-token", "utf8").trim();
	if (!t) throw new Error("AI 代號是空的");
	return t;
}
/** 依環境變數建立平台 client：PT_API_URL、PT_KEY_ID、PT_KEY_FILE（預設 ../keys/server.key）或 PT_PRIVATE_KEY、PT_MOCK=1 */
function platformFromEnv() {
	if (process.env.PT_MOCK === "1") return createServerClient({
		baseUrl: "",
		keyId: "",
		mock: true
	});
	const baseUrl = process.env.PT_API_URL || "";
	const keyId = process.env.PT_KEY_ID || "";
	if (!baseUrl || !keyId) throw new Error("請設定 PT_API_URL 與 PT_KEY_ID（本機開發用 pt dev 會自動設定）");
	const raw = (process.env.PT_PRIVATE_KEY || readFileSync(process.env.PT_KEY_FILE || "../keys/server.key", "utf8")).trim();
	const seed = Buffer.from(raw, "base64");
	if (seed.length !== 32) throw new Error("伺服器私鑰格式錯誤：請用 pt keygen -purpose server 產生");
	return createServerClient({
		baseUrl,
		keyId,
		privateKey: createPrivateKey({
			key: Buffer.concat([Buffer.from("302e020100300506032b657004220420", "hex"), seed]),
			format: "der",
			type: "pkcs8"
		})
	});
}
function originAllowed(allowed, origin) {
	return allowed.some((raw) => {
		const a = raw.trim();
		if (!a) return false;
		if (a === origin) return true;
		const i = a.indexOf("://*.");
		if (i <= 0) return false;
		const prefix = a.slice(0, i) + "://";
		const suffix = "." + a.slice(i + 5);
		if (!origin.startsWith(prefix) || !origin.endsWith(suffix)) return false;
		const sub = origin.slice(prefix.length, origin.length - suffix.length);
		return sub.length > 0 && !/[/:@]/.test(sub);
	});
}
function createLimiter(rate, burst) {
	const buckets = /* @__PURE__ */ new Map();
	let swept = 0;
	return (key, now) => {
		if (now - swept > 6e4) {
			for (const [k, b] of buckets) if (now - b.at > 6e5) buckets.delete(k);
			swept = now;
		}
		let b = buckets.get(key);
		if (!b) {
			b = {
				tokens: burst,
				at: now
			};
			buckets.set(key, b);
		}
		if (now > b.at) b.tokens = Math.min(burst, b.tokens + (now - b.at) / 1e3 * rate);
		b.at = now;
		if (b.tokens < 1) return false;
		b.tokens -= 1;
		return true;
	};
}
var TOO_LARGE = Symbol("too_large");
function createGameServer(opts) {
	const log = opts.log ?? ((...a) => console.warn(...a));
	const now = opts.now ?? Date.now;
	const seatHold = positive(opts.seatHoldMs, 0) || envDuration("SEAT_HOLD", 6e4, log);
	const roomSync = positive(opts.roomSyncMs, 0) || envDuration("ROOM_SYNC", 1e4, log);
	const tickMs = positive(opts.tickMs, 200);
	const statusSync = positive(opts.statusSyncMs, 0) || envDuration("STATUS_SYNC", 6e4, log);
	const allowed = opts.allowOrigins ?? (process.env.ALLOW_ORIGINS || "").split(",").map((s) => s.trim()).filter(Boolean);
	const trustProxy = positive(opts.trustProxy, 0) || positive(Number.parseInt(process.env.TRUST_PROXY || "0", 10), 0);
	const maxBody = positive(opts.maxBody, 1024);
	const l = opts.limits ?? {};
	const actionLimit = createLimiter(positive(l.actionRate, 20), positive(l.actionBurst, 40));
	const ipLimit = createLimiter(positive(l.ipRate, 60), positive(l.ipBurst, 120));
	const joinLimit = createLimiter(positive(l.joinRate, .5), positive(l.joinBurst, 10));
	const pt = opts.platform;
	const rooms = /* @__PURE__ */ new Map();
	const tokens = /* @__PURE__ */ new Map();
	let syncFailingSince = 0;
	let seatSeq = 0;
	let stopReason = "";
	const queue = [];
	let sending = false;
	let stopped = false;
	let idleWaiters = [];
	const retryable = (e) => !(e instanceof PlatformApiError) || e.status >= 500 || e.status === 429;
	async function pump() {
		if (sending) return;
		sending = true;
		try {
			for (let r = queue.shift(); r && !stopped; r = queue.shift()) {
				const what = r.result ? `結束（${r.result}）` : "開始";
				for (let attempt = 0;; attempt++) try {
					if (r.result) await pt.reportRoundEnd(r.roomId, r.epoch, r.roundId, r.result);
					else await pt.reportRoundStart(r.roomId, r.epoch, r.roundId, r.playerIds);
					break;
				} catch (e) {
					if (!retryable(e) || attempt >= retryDelaysMs.length || stopped) {
						log(`回報一局${what}失敗（不影響遊戲）：`, e.message);
						break;
					}
					await new Promise((ok) => setTimeout(ok, retryDelaysMs[attempt]));
					if (stopped) break;
				}
			}
		} finally {
			sending = false;
			const w = idleWaiters;
			idleWaiters = [];
			for (const f of w) f();
		}
	}
	function push(r) {
		if (stopped) return;
		if (queue.length >= MAX_QUEUED_REPORTS) queue.shift();
		queue.push(r);
		pump();
	}
	function makeRoom(id, epoch) {
		let roundId = "";
		let rounds = 0;
		const room = {
			id,
			epoch,
			seatMap: /* @__PURE__ */ new Map(),
			game: void 0,
			emptyAt: 0,
			now,
			seats() {
				const t = now();
				return [...room.seatMap.values()].sort((a, b) => a.order - b.order).map((s) => ({
					playerId: s.playerId,
					name: s.name,
					online: t - s.lastSeen < OFFLINE_AFTER_MS
				}));
			},
			round: () => roundId,
			startRound(playerIds) {
				if (roundId) throw new ActionError("round_active");
				if (!playerIds.length) throw new ActionError("no_players");
				rounds += 1;
				roundId = `e${epoch}-r${rounds}-${newToken().slice(0, 6)}`;
				push({
					roomId: id,
					epoch,
					roundId,
					playerIds: [...playerIds]
				});
				return roundId;
			},
			endRound(result) {
				if (!roundId) return;
				push({
					roomId: id,
					epoch,
					roundId,
					result
				});
				roundId = "";
			}
		};
		return room;
	}
	const safely = (room, what, fn) => {
		try {
			fn();
		} catch (e) {
			log(`房間 ${room.id} 的 ${what} 發生錯誤：`, e);
		}
	};
	function removeSeat(room, playerId) {
		const s = room.seatMap.get(playerId);
		if (!s) return;
		tokens.delete(s.token);
		room.seatMap.delete(playerId);
		safely(room, "leave", () => room.game.leave(playerId));
		if (room.seatMap.size === 0) room.emptyAt = now();
	}
	function endRoom(room) {
		for (const s of room.seatMap.values()) tokens.delete(s.token);
		rooms.delete(room.id);
		room.endRound("aborted");
		safely(room, "close", () => room.game.close());
	}
	class HttpError extends Error {
		status;
		code;
		constructor(status, code) {
			super(code);
			this.status = status;
			this.code = code;
		}
	}
	async function join(code) {
		if (stopReason) throw new HttpError(503, stopReason);
		if (syncFailingSince && now() - syncFailingSince > STALE_SYNCS * roomSync) throw new HttpError(503, "platform_unavailable");
		let ex;
		try {
			ex = await pt.exchangeLaunchCode(code);
		} catch (e) {
			if (e instanceof PlatformApiError && e.status < 500 && e.status !== 429) throw new HttpError(403, e.code);
			log("兌換啟動碼失敗：", e.message);
			throw new HttpError(502, "unavailable");
		}
		if (stopReason) throw new HttpError(503, stopReason);
		let room = rooms.get(ex.roomId);
		if (room && ex.epoch < room.epoch) throw new HttpError(409, "stale_epoch");
		let newRoom = false;
		if (!room || ex.epoch > room.epoch) {
			if (room) endRoom(room);
			const nr = makeRoom(ex.roomId, ex.epoch);
			nr.game = opts.newGame(nr);
			room = nr;
			newRoom = true;
			rooms.set(room.id, room);
		}
		let s = room.seatMap.get(ex.playerId);
		const newSeat = !s;
		if (!s) {
			seatSeq += 1;
			s = {
				playerId: ex.playerId,
				name: ex.name,
				token: "",
				lastSeen: 0,
				order: seatSeq
			};
			room.seatMap.set(s.playerId, s);
		}
		const ref = {
			roomId: room.id,
			playerId: s.playerId
		};
		const prev = {
			token: s.token,
			lastSeen: s.lastSeen
		};
		const token = newToken();
		tokens.delete(prev.token);
		s.token = token;
		s.lastSeen = now();
		tokens.set(token, ref);
		try {
			return JSON.stringify({
				token,
				playerId: s.playerId,
				name: s.name,
				state: room.game.view(room, s.playerId)
			});
		} catch (e) {
			tokens.delete(token);
			if (newRoom) {
				rooms.delete(room.id);
				const r = room;
				safely(r, "close", () => r.game.close());
			} else if (newSeat) {
				room.seatMap.delete(s.playerId);
				if (room.seatMap.size === 0) room.emptyAt = now();
			} else {
				s.token = prev.token;
				s.lastSeen = prev.lastSeen;
				if (prev.token) tokens.set(prev.token, ref);
			}
			throw e;
		}
	}
	function action(token, act, data) {
		const t = typeof token === "string" ? tokens.get(token) : void 0;
		const room = t && rooms.get(t.roomId);
		const s = t && room?.seatMap.get(t.playerId);
		if (!room || !s || s.token !== token) throw new HttpError(401, "session_expired");
		if (typeof act !== "string") throw new HttpError(400, "bad_request");
		s.lastSeen = now();
		const ret = opts.onAction(room, {
			playerId: s.playerId,
			name: s.name,
			online: true
		}, room.game, act, data);
		if (ret && typeof ret.then === "function") {
			log("onAction 不能是 async（回報一局請用 room.startRound／endRound）");
			throw new HttpError(500, "internal");
		}
		if (rooms.get(room.id) !== room) throw new HttpError(401, "session_expired");
		return JSON.stringify(room.game.view(room, s.playerId));
	}
	function sweep() {
		const t = now();
		for (const room of [...rooms.values()]) {
			for (const [id, s] of [...room.seatMap]) {
				if (t - s.lastSeen <= seatHold) continue;
				let free = false;
				safely(room, "canFree", () => free = room.game.canFree(id));
				if (free) removeSeat(room, id);
			}
			if (room.seatMap.size === 0 && room.emptyAt && t - room.emptyAt > IDLE_ROOM_MS) endRoom(room);
		}
	}
	async function syncRooms() {
		const list = [...rooms.values()].map((r) => ({
			id: r.id,
			epoch: r.epoch
		}));
		let failed = false;
		for (const { id, epoch } of list) {
			let m = null;
			let gone = false;
			try {
				m = await pt.getRoomMembers(id);
			} catch (e) {
				gone = e instanceof PlatformApiError && e.status === 404;
				if (!gone) {
					log("查詢房間失敗（下次再試）：", e.message);
					failed = true;
					continue;
				}
			}
			const room = rooms.get(id);
			if (!room || room.epoch !== epoch) continue;
			if (gone || !m.open || m.epoch > room.epoch) {
				endRoom(room);
				continue;
			}
			for (const mem of m.members || []) if (mem.kicked) removeSeat(room, mem.playerId);
		}
		if (!list.length) return;
		if (!failed) syncFailingSince = 0;
		else if (!syncFailingSince) syncFailingSince = now();
	}
	function stopServing(reason) {
		if (stopReason === reason) return;
		stopReason = reason;
		const n = rooms.size;
		for (const room of [...rooms.values()]) endRoom(room);
		log(`停止服務（${reason}）：結束 ${n} 個房間，拒絕新加入；/readyz 回 503。玩聚恢復這款遊戲後會自動恢復`);
	}
	async function checkStatus() {
		if (!pt.getGameStatus) return;
		try {
			const st = await pt.getGameStatus();
			if (!st.serving) stopServing(st.reason || "game_disabled");
			else if (stopReason) {
				log(`恢復服務（之前：${stopReason}）`);
				stopReason = "";
			}
		} catch (e) {
			if (e instanceof PlatformApiError && e.status === 401 && e.code === "unknown_key") stopServing("key_revoked");
			else log("查詢遊戲狀態失敗（下次再試，不影響服務）：", e.message);
		}
	}
	function tick() {
		for (const room of [...rooms.values()]) {
			const g = room.game;
			if (g.tick) safely(room, "tick", () => g.tick(room));
		}
	}
	const fromProxy = (addr) => {
		const a = addr.replace(/^::ffff:/, "");
		return a === "::1" || /^127\./.test(a) || /^10\./.test(a) || /^192\.168\./.test(a) || /^169\.254\./.test(a) || /^fe[89ab][0-9a-f]:/i.test(a) || /^172\.(1[6-9]|2\d|3[01])\./.test(a) || /^f[cd][0-9a-f]{2}:/i.test(a);
	};
	const clientIP = (req) => {
		const remote = req.socket.remoteAddress || "";
		if (trustProxy > 0 && fromProxy(remote)) {
			const xff = req.headers["x-forwarded-for"];
			const parts = (Array.isArray(xff) ? xff.join(",") : xff || "").split(",").filter((p) => p.trim());
			if (parts.length >= trustProxy) return (parts[parts.length - trustProxy] ?? "").trim();
		}
		return remote;
	};
	const readBody = (req) => new Promise((resolve) => {
		const chunks = [];
		let size = 0;
		let over = false;
		req.on("data", (c) => {
			if (over) return;
			size += c.length;
			if (size > maxBody) {
				over = true;
				chunks.length = 0;
				resolve(TOO_LARGE);
				return;
			}
			chunks.push(c);
		});
		req.on("end", () => {
			if (over) return;
			try {
				const v = JSON.parse(Buffer.concat(chunks).toString("utf8"));
				resolve(v && typeof v === "object" && !Array.isArray(v) ? v : null);
			} catch {
				resolve(null);
			}
		});
		req.on("error", () => resolve(null));
	});
	const sendRaw = (res, status, json, extra = {}) => {
		res.writeHead(status, {
			"Content-Type": "application/json",
			...extra
		});
		res.end(json);
	};
	const send = (res, status, body, extra = {}) => sendRaw(res, status, JSON.stringify(body), extra);
	async function handle(req, res) {
		const path = (req.url || "/").split("?")[0];
		const api = path === "/join" || path === "/action" || path === "/healthz" || path === "/readyz";
		const origin = req.headers.origin;
		res.setHeader("Vary", "Origin");
		if (origin && originAllowed(allowed, origin)) {
			res.setHeader("Access-Control-Allow-Origin", origin);
			res.setHeader("Access-Control-Allow-Methods", "GET, POST, OPTIONS");
			res.setHeader("Access-Control-Allow-Headers", "Content-Type");
			res.setHeader("Access-Control-Max-Age", "600");
		}
		if (req.method === "OPTIONS") {
			res.writeHead(204);
			res.end();
			return true;
		}
		if (!api) return false;
		if (req.method === "GET" && path === "/healthz") {
			send(res, 200, { status: "ok" });
			return true;
		}
		if (req.method === "GET" && path === "/readyz") {
			if (stopReason) send(res, 503, {
				status: "stopped",
				reason: stopReason
			});
			else send(res, 200, { status: "ok" });
			return true;
		}
		if (req.method !== "POST" || path === "/healthz" || path === "/readyz") {
			send(res, 405, { error: "method_not_allowed" });
			return true;
		}
		const ip = clientIP(req);
		const t = now();
		if (!ipLimit(ip, t) || path === "/join" && !joinLimit(ip, t)) {
			send(res, 429, { error: "rate_limited" });
			return true;
		}
		const body = await readBody(req);
		if (body === TOO_LARGE) {
			send(res, 413, { error: "too_large" }, { Connection: "close" });
			return true;
		}
		if (!body) {
			send(res, 400, { error: "bad_request" });
			return true;
		}
		try {
			if (path === "/join") {
				if (typeof body.launchCode !== "string" || !body.launchCode) throw new HttpError(400, "bad_request");
				sendRaw(res, 200, await join(body.launchCode));
			} else {
				if (!actionLimit(String(body.token ?? ""), now())) throw new HttpError(429, "rate_limited");
				sendRaw(res, 200, action(body.token, body.action, body.data));
			}
		} catch (e) {
			if (e instanceof HttpError) send(res, e.status, { error: e.code });
			else if (e instanceof ActionError) send(res, 400, { error: e.code });
			else {
				log("處理請求時發生錯誤：", e);
				send(res, 500, { error: "internal" });
			}
		}
		return true;
	}
	const timers = [];
	let syncing = false;
	return {
		handle,
		start() {
			if (pt.getGameStatus) {
				let checking = false;
				const run = async () => {
					if (checking) return;
					checking = true;
					try {
						await checkStatus();
					} finally {
						checking = false;
					}
				};
				run();
				timers.push(setInterval(run, statusSync));
			}
			timers.push(setInterval(sweep, 1e3));
			timers.push(setInterval(tick, tickMs));
			timers.push(setInterval(async () => {
				if (syncing) return;
				syncing = true;
				try {
					await syncRooms();
				} finally {
					syncing = false;
				}
			}, roomSync));
		},
		async close(timeoutMs = 5e3) {
			for (const t of timers) clearInterval(t);
			if (sending || queue.length > 0) {
				let timer;
				await Promise.race([new Promise((ok) => idleWaiters.push(ok)), new Promise((ok) => timer = setTimeout(ok, timeoutMs))]);
				clearTimeout(timer);
			}
			stopped = true;
		},
		listen(port, host, fallback) {
			return createServer(async (req, res) => {
				try {
					if (await handle(req, res)) return;
					if (fallback) await fallback(req, res);
					else {
						res.writeHead(404);
						res.end("not found");
					}
				} catch (e) {
					log("處理請求時發生錯誤：", e);
					if (!res.headersSent) res.writeHead(400);
					res.end();
				}
			}).listen(port, host);
		},
		stopped: () => stopReason,
		internal: {
			sweep,
			syncRooms,
			tick,
			checkStatus,
			stopServing,
			rooms
		}
	};
}
// src/index.ts
var SIGNATURE_SCHEME = "PT-SIG-1";
var MAX_CLOCK_SKEW_MS = 3e5;
var HEADER = {
	keyId: "x-pt-key-id",
	timestamp: "x-pt-timestamp",
	nonce: "x-pt-nonce",
	signature: "x-pt-signature",
	idempotencyKey: "idempotency-key"
};
function canonicalQuery(query) {
	if (!query) return "";
	const pairs = [];
	for (const [k, v] of Object.entries(query)) for (const value of Array.isArray(v) ? v : [v]) pairs.push([encodeURIComponent(k), encodeURIComponent(value)]);
	pairs.sort(([ak, av], [bk, bv]) => ak === bk ? av < bv ? -1 : av > bv ? 1 : 0 : ak < bk ? -1 : 1);
	return pairs.map(([k, v]) => `${k}=${v}`).join("&");
}
/** 簽章字串：方法、路徑、查詢、時間戳記、nonce、內容雜湊，每項一行 */
function canonicalString(req, timestamp, nonce) {
	const bodyHash = createHash("sha256").update(req.body, "utf8").digest("base64url");
	return [
		SIGNATURE_SCHEME,
		req.method.toUpperCase(),
		req.path,
		canonicalQuery(req.query),
		timestamp,
		nonce,
		bodyHash
	].join("\n");
}
function signRequest(req, credentials, options = {}) {
	const timestamp = String(options.now ?? Date.now());
	const nonce = options.nonce ?? randomBytes(16).toString("base64url");
	const signature = sign(null, Buffer.from(canonicalString(req, timestamp, nonce)), credentials.privateKey);
	return {
		[HEADER.keyId]: credentials.keyId,
		[HEADER.timestamp]: timestamp,
		[HEADER.nonce]: nonce,
		[HEADER.signature]: signature.toString("base64url")
	};
}
async function verifyRequest(req, deps) {
	const h = Object.fromEntries(Object.entries(req.headers).map(([k, v]) => [k.toLowerCase(), v]));
	const keyId = h[HEADER.keyId];
	const timestamp = h[HEADER.timestamp];
	const nonce = h[HEADER.nonce];
	const signature = h[HEADER.signature];
	if (!keyId || !timestamp || !nonce || !signature) return {
		ok: false,
		error: "missing_headers"
	};
	if (!/^\d{1,16}$/.test(timestamp)) return {
		ok: false,
		error: "bad_timestamp"
	};
	const now = deps.now ?? Date.now();
	if (Math.abs(now - Number(timestamp)) > 3e5) return {
		ok: false,
		error: "clock_skew"
	};
	const publicKey = await deps.getPublicKey(keyId);
	if (!publicKey) return {
		ok: false,
		error: "unknown_key"
	};
	let valid = false;
	try {
		valid = verify(null, Buffer.from(canonicalString(req, timestamp, nonce)), publicKey, Buffer.from(signature, "base64url"));
	} catch {
		valid = false;
	}
	if (!valid) return {
		ok: false,
		error: "bad_signature"
	};
	if (!await deps.nonceStore.useOnce(keyId, nonce, 6e5)) return {
		ok: false,
		error: "replay"
	};
	return {
		ok: true,
		keyId
	};
}
var PlatformApiError = class extends Error {
	status;
	code;
	constructor(status, code) {
		super(`玩聚平台 API 錯誤：${status} ${code}`);
		this.status = status;
		this.code = code;
		this.name = "PlatformApiError";
	}
};
/** 瀏覽器 SDK 模擬模式產生的啟動碼前綴 */
var MOCK_CODE_PREFIX = "mock-";
function createServerClient(options) {
	if (options.mock) return mockClient();
	if (!options.privateKey) throw new Error("需要 privateKey（遊戲伺服器金鑰的私鑰）；本機開發可改用 mock: true");
	const credentials = {
		keyId: options.keyId,
		privateKey: options.privateKey
	};
	const doFetch = options.fetch ?? fetch;
	const base = options.baseUrl.replace(/\/+$/, "");
	const timeoutMs = options.timeoutMs && options.timeoutMs > 0 ? options.timeoutMs : 1e4;
	async function request(method, path, data, idempotencyKey, query) {
		const body = data === void 0 ? "" : JSON.stringify(data);
		const headers = { ...signRequest({
			method,
			path,
			body,
			...query ? { query } : {}
		}, credentials) };
		if (data !== void 0) headers["content-type"] = "application/json";
		if (idempotencyKey) headers[HEADER.idempotencyKey] = idempotencyKey;
		const qs = query ? "?" + new URLSearchParams(query).toString() : "";
		const res = await doFetch(`${base}${path}${qs}`, {
			method,
			headers,
			signal: AbortSignal.timeout(timeoutMs),
			...method === "POST" ? { body } : {}
		});
		const json = await res.json().catch(() => null);
		if (!res.ok) {
			const code = json && typeof json === "object" && "error" in json ? String(json.error) : "unknown";
			throw new PlatformApiError(res.status, code);
		}
		return json;
	}
	const post = (path, data, idempotencyKey) => request("POST", path, data, idempotencyKey);
	return {
		/** 兌換一次性啟動碼，取得玩家假名 ID、暱稱、平台房號 */
		async exchangeLaunchCode(code) {
			const json = await post("/v1/launch/exchange", { code });
			return launchExchangeResponseSchema.parse(json);
		},
		/**
		* 查詢房間狀態：是否開放、局數（房主重新開始時加一）、每位成員是否被房主踢出。
		* 建議定期查詢（例如每 10 秒），把被踢的玩家斷線、局數改變時開新局。
		*/
		async getRoomMembers(roomId) {
			const json = await request("GET", `/v1/game/rooms/${encodeURIComponent(roomId)}/members`);
			if (!json || typeof json.epoch !== "number" || !Array.isArray(json.members)) throw new PlatformApiError(502, "bad_response");
			return json;
		},
		/**
		* 回報一局開始與這一局的玩家（只用於平台的成效分析）。roundId 由遊戲決定，同一房間內不可重複；
		* epoch 必須是目前的局數，舊局數會被拒絕（stale_epoch）。同一局重送不會重複記錄。
		*/
		async reportRoundStart(roomId, epoch, roundId, playerIds) {
			if (!/^[A-Za-z0-9_.:-]{1,64}$/.test(roundId) || playerIds.length === 0) throw new PlatformApiError(400, "bad_request");
			await post("/v1/game/rounds", {
				roomId,
				epoch,
				roundId,
				phase: "start",
				playerIds
			}, `round:${roomId}:${roundId}:start`);
		},
		/** 回報一局結束：completed（正常玩完）或 aborted（中斷） */
		async reportRoundEnd(roomId, epoch, roundId, result) {
			await post("/v1/game/rounds", {
				roomId,
				epoch,
				roundId,
				phase: "end",
				result
			}, `round:${roomId}:${roundId}:end`);
		},
		async getGameStatus() {
			const json = await request("GET", "/v1/game/status");
			if (!json || typeof json.serving !== "boolean") throw new PlatformApiError(502, "bad_response");
			return json;
		},
		/** 送出發言規則；同一房間同一序號重送不會重複執行 */
		async voicePolicy(policy) {
			const checked = voicePolicySchema.parse(policy);
			await post("/v1/voice/policy", checked, `voice:${checked.roomId}:${checked.sequence}`);
		},
		async pitchPolicy(policy) {
			const checked = pitchPolicySchema.parse(policy);
			await post("/v1/pitch/policy", checked, `pitch:${checked.roomId}:${checked.sequence}`);
		},
		async createPurchase(req) {
			if (!req.roomId || !req.playerId || !req.sku || !/^[A-Za-z0-9._:-]{8,100}$/.test(req.idempotencyKey)) throw new PlatformApiError(400, "bad_request");
			const json = await post("/v1/game/purchases", {
				roomId: req.roomId,
				playerId: req.playerId,
				sku: req.sku,
				quantity: 1,
				idempotencyKey: req.idempotencyKey
			});
			if (!json || typeof json.purchaseId !== "string") throw new PlatformApiError(502, "bad_response");
			return json;
		},
		async getPurchase(purchaseId) {
			const json = await request("GET", `/v1/game/purchases/${encodeURIComponent(purchaseId)}`);
			if (!json || typeof json.status !== "string") throw new PlatformApiError(502, "bad_response");
			return json;
		},
		async getRoomEntitlements(roomId) {
			return (await request("GET", `/v1/game/rooms/${encodeURIComponent(roomId)}/entitlements`))?.players ?? {};
		},
		async listPurchaseEvents(after) {
			const json = await request("GET", "/v1/game/purchase-events", void 0, void 0, { after: String(Math.max(0, Math.trunc(after))) });
			return {
				events: json?.events ?? [],
				next: typeof json?.next === "number" ? json.next : after
			};
		}
	};
}
/** 模擬模式的客戶端：同一個啟動碼對應同一位假玩家（方便測試重新加入），名稱帶代號方便分辨多個分頁 */
function mockClient() {
	return {
		async exchangeLaunchCode(code) {
			if (!code.startsWith("mock-") || code.length < 13) throw new PlatformApiError(409, "invalid_code");
			const suffix = code.slice(5, 13);
			const playerId = "mock-p-" + suffix;
			return {
				gameId: "mock-game",
				playerId,
				name: "測試玩家-" + suffix.slice(0, 4).toUpperCase(),
				roomId: "mock-room",
				epoch: 0,
				hostPlayerId: playerId
			};
		},
		async getRoomMembers(roomId) {
			return {
				roomId,
				open: true,
				epoch: 0,
				hostPlayerId: "",
				members: []
			};
		},
		async getGameStatus() {
			return {
				gameId: "mock-game",
				status: "live",
				serving: true,
				reason: "",
				maintenance: {
					game: false,
					site: ""
				}
			};
		},
		async reportRoundStart() {},
		async reportRoundEnd() {},
		async voicePolicy() {},
		async pitchPolicy(policy) {
			pitchPolicySchema.parse(policy);
		},
		async createPurchase() {
			throw new PlatformApiError(503, "points_disabled");
		},
		async getPurchase() {
			throw new PlatformApiError(503, "points_disabled");
		},
		async getRoomEntitlements() {
			return {};
		},
		async listPurchaseEvents(after) {
			return {
				events: [],
				next: after
			};
		}
	};
}
export { ActionError, HEADER, MAX_CLOCK_SKEW_MS, MOCK_CODE_PREFIX, PlatformApiError, SIGNATURE_SCHEME, aiBaseURL, aiToken, canonicalString, createGameServer, createServerClient, platformFromEnv, retryDelaysMs, signRequest, verifyRequest };
