// ../../node_modules/zod/v4/core/util.js
function e(e) {
	let t = Object.values(e).filter((e) => typeof e == "number");
	return Object.entries(e).filter(([e, n]) => t.indexOf(+e) === -1).map(([e, t]) => t);
}
function t(e, t) {
	return typeof t == "bigint" ? t.toString() : t;
}
var n = class {
	constructor(e) {
		this._getter = e, this._value = void 0;
	}
	get value() {
		let e = this._getter;
		return e !== void 0 && (this._value = e(), this._getter = void 0), this._value;
	}
};
function r(e) {
	return new n(e);
}
function i(e) {
	return e == null;
}
function a(e) {
	let t = +!!e.startsWith("^"), n = e.endsWith("$") ? e.length - 1 : e.length;
	return e.slice(t, n);
}
function o(e, t, n) {
	Object.defineProperty(e, t, {
		value: n,
		writable: !0,
		enumerable: !0,
		configurable: !0
	});
}
var s = "captureStackTrace" in Error ? Error.captureStackTrace : (...e) => {};
function c(e) {
	return typeof e == "object" && !!e && !Array.isArray(e);
}
function l(e) {
	if (c(e) === !1) return !1;
	let t = e.constructor;
	if (t === void 0 || typeof t != "function") return !0;
	let n = t.prototype;
	return c(n) !== !1 && Object.prototype.hasOwnProperty.call(n, "isPrototypeOf") !== !1;
}
function u(e) {
	return l(e) ? { ...e } : Array.isArray(e) ? [...e] : e instanceof Map ? new Map(e) : e instanceof Set ? new Set(e) : e;
}
var d = /* @__PURE__*/ new Set([
	"string",
	"number",
	"symbol"
]);
function f(e) {
	return e.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}
function p(e, t, n) {
	let r = new e._zod.constr(t ?? e._zod.def);
	return (!t || n?.parent) && (r._zod.parent = e), r;
}
function m(e) {
	let t = e;
	if (!t) return {};
	if (typeof t == "string") return { error: () => t };
	if (t?.message !== void 0) {
		if (t?.error !== void 0) throw Error("Cannot specify both `message` and `error` params");
		t.error = t.message;
	}
	return delete t.message, typeof t.error == "string" ? {
		...t,
		error: () => t.error
	} : t;
}
function h(e) {
	return Object.keys(e).filter((t) => e[t]._zod.optin !== void 0 && e[t]._zod.optout === "optional");
}
var ee = {
	safeint: [-(2 ** 53 - 1), 2 ** 53 - 1],
	int32: [-2147483648, 2147483647],
	uint32: [0, 4294967295],
	float32: [-34028234663852886e22, 34028234663852886e22],
	float64: [-Number.MAX_VALUE, Number.MAX_VALUE]
};
function g(e, t = 0) {
	if (e.aborted === !0) return !0;
	for (let n = t; n < e.issues.length; n++) if (e.issues[n]?.continue !== !0) return !0;
	return !1;
}
function te(e, t = 0) {
	if (e.aborted === !0) return !0;
	for (let n = t; n < e.issues.length; n++) if (e.issues[n]?.continue === !1) return !0;
	return !1;
}
function ne(e, t) {
	return t.map((t) => {
		var n;
		return (n = t).path ?? (n.path = []), t.path.unshift(e), t;
	});
}
function _(e) {
	return typeof e == "string" ? e : e?.message;
}
function re(e) {
	let t = [];
	for (let n of e ?? []) {
		let e = n._zod.def;
		if (!/_(length|size)$|_equals$/.test(e.check)) break;
		let r = e.maximum ?? e.length ?? e.size;
		r >= 0 && t.push([r, n]);
	}
	return t;
}
function v(e, t, n) {
	var r;
	for (let i = t; i < e.length; i++) (r = e[i]).schema ?? (r.schema = n);
}
function y(e, t, n) {
	var r;
	let i = e.inst?._zod?.traits;
	i?.has("$ZodType") && (i.has("$ZodCheck") ? (r = e).schema ?? (r.schema = e.inst) : e.schema = e.inst);
	let a = e.schema === e.inst ? void 0 : e.schema?._zod.def?.error, o = e.message ? e.message : _(e.inst?._zod.def?.error?.(e)) ?? _(a?.(e)) ?? _(t?.error?.(e)) ?? _(n.customError?.(e)) ?? _(n.localeError?.(e)) ?? "Invalid input", s = {};
	for (let t of Object.keys(e)) t !== "inst" && t !== "schema" && t !== "continue" && t !== "input" && t !== "__proto__" && (s[t] = e[t]);
	return s.path ?? (s.path = []), s.message = o, t?.reportInput && (s.input = e.input), s;
}
var ie = /[\uD800-\uDBFF]/;
function b(e) {
	let t = e.length;
	if (!ie.test(e)) return t;
	let n = t;
	for (let r = 0; r < t - 1; r++) (e.charCodeAt(r) & 64512) == 55296 && (e.charCodeAt(r + 1) & 64512) == 56320 && (n--, r++);
	return n;
}
function ae(e) {
	return Array.isArray(e) ? "array" : typeof e == "string" ? "string" : "unknown";
}
function oe(...e) {
	let [t, n, r] = e;
	return typeof t == "string" ? {
		message: t,
		code: "custom",
		input: n,
		inst: r
	} : { ...t };
}
function se(e, t) {
	for (let n in t) {
		let r = Object.getOwnPropertyDescriptor(t, n);
		r.get ? Object.defineProperty(e, n, {
			...r,
			enumerable: !1
		}) : le(e, n, r.value);
	}
}
function x(e, t, n, r = !0) {
	return Object.defineProperty(e, t, {
		configurable: !0,
		writable: !0,
		enumerable: r,
		value: n
	}), n;
}
function ce(e, t, n) {
	return x(e, t, n, !1);
}
function le(e, t, n) {
	Object.defineProperty(e, t, {
		configurable: !0,
		get() {
			return this == null ? n : x(this, t, n.bind(this));
		},
		set(e) {
			x(this, t, e);
		}
	});
}
function ue(e, t) {
	let n = Object.getPrototypeOf(e);
	return t in n ? void 0 : n;
}
var S, C = !1, de = {
	configurable: !0,
	get() {
		C = !0;
	}
};
function w(e, t, n) {
	let r = Object.getPrototypeOf(e._zod);
	t in r && S !== e._zod ? S = void 0 : (S = e._zod, Object.defineProperty(r, t, {
		configurable: !0,
		get() {
			Object.defineProperty(this, t, de);
			let e = C;
			C = !1;
			try {
				let r = n(this);
				return C ? delete this[t] : Object.defineProperty(this, t, {
					configurable: !0,
					writable: !0,
					value: r
				}), C = C || e, r;
			} catch (n) {
				throw delete this[t], C = C || e, n;
			}
		},
		set(e) {
			Object.defineProperty(this, t, {
				configurable: !0,
				writable: !0,
				value: e
			});
		}
	}));
}
function fe(e, t, n, r) {
	let i = ue(e, t);
	i && Object.defineProperty(i, t, {
		configurable: !0,
		get() {
			let e = {
				configurable: !0,
				writable: !0,
				enumerable: r,
				value: void 0
			};
			return Object.defineProperty(this, t, e), e.value = n(this), Object.defineProperty(this, t, e), e.value;
		},
		set(e) {
			Object.defineProperty(this, t, {
				configurable: !0,
				writable: !0,
				enumerable: r,
				value: e
			});
		}
	});
}
// ../../node_modules/zod/v4/core/core.js
var pe, T = {
	value: void 0,
	enumerable: !1
}, E = "captureStackTrace" in Error ? Error : null;
function me(e) {
	let t = E;
	if (t) {
		let n = t.stackTraceLimit;
		if (typeof n == "number") {
			try {
				t.stackTraceLimit = 0;
			} catch {
				return E = null, new e();
			}
			try {
				return new e();
			} finally {
				t.stackTraceLimit = n;
			}
		}
	}
	return new e();
}
/* @__NO_SIDE_EFFECTS__ */
function D(e, t, n, r) {
	let i = {};
	function a(e) {
		this.def = e, this.constr = d, this.traits = /* @__PURE__ */ new Set();
	}
	a.prototype = i;
	let o = n, s = o && /* @__PURE__ */ new WeakSet();
	function c(n, r) {
		if (!n._zod) {
			T.value = new a(r);
			try {
				Object.defineProperty(n, "_zod", T);
			} finally {
				T.value = void 0;
			}
		} else if (n._zod.traits.has(e)) return;
		if (n._zod.traits.add(e), t(n, r), s) {
			let e = Object.getPrototypeOf(n), t = n._zod.constr.prototype, r = e;
			for (; r && r !== t;) r = Object.getPrototypeOf(r);
			let i = r ?? e;
			s.has(i) || (s.add(i), se(i, o));
		}
		let i = d.prototype;
		for (let e in i) Object.prototype.hasOwnProperty.call(i, e) && (e in n || (n[e] = i[e].bind(n)));
	}
	let l = r?.Parent ?? Object;
	class u extends l {}
	Object.defineProperty(u, "name", { value: e });
	function d(e) {
		let t = r?.Parent ? me(u) : this;
		c(t, e);
		let n = t._zod.deferred;
		if (n) {
			for (let e of n) e();
			t._zod.deferred = void 0;
		}
		let i = globalThis.__zod_globalConfig?.postProcessor;
		return i && i(t), t;
	}
	return Object.defineProperty(d, "init", { value: c }), Object.defineProperty(d, Symbol.hasInstance, { value: (t) => r?.Parent && t instanceof r.Parent ? !0 : t?._zod?.traits?.has(e) }), Object.defineProperty(d, "name", { value: e }), d;
}
var O = class extends Error {
	constructor() {
		super("Encountered Promise during synchronous parse. Use .parseAsync() instead.");
	}
};
(pe = globalThis).__zod_globalConfig ?? (pe.__zod_globalConfig = {});
var k = globalThis.__zod_globalConfig;
function A(e) {
	return e && Object.assign(k, e), k;
}
// ../../node_modules/zod/v4/core/errors.js
function he() {
	let e = this._zod;
	return e.message ?? (e.message = JSON.stringify(e.def, t, 2)), e.message;
}
function ge(e) {
	this._zod.message = e;
}
var _e = {
	get: he,
	set: ge,
	enumerable: !0,
	configurable: !0
}, j = {
	value: void 0,
	enumerable: !1
}, ve = /* @__PURE__ */ new WeakSet([Object.prototype, Error.prototype]), M = /* @__PURE__ */ D("$ZodError", (e, t) => {
	e.name = "$ZodError", j.value = t, Object.defineProperty(e, "issues", j), j.value = void 0, Object.defineProperty(e, "message", _e);
	let n = Object.getPrototypeOf(e);
	ve.has(n) || (ve.add(n), Object.defineProperty(n, "toString", {
		configurable: !0,
		enumerable: !1,
		get() {
			let e = () => this.message;
			return Object.defineProperty(this, "toString", {
				value: e,
				configurable: !0,
				writable: !0
			}), e;
		},
		set(e) {
			Object.defineProperty(this, "toString", {
				value: e,
				configurable: !0,
				writable: !0
			});
		}
	}));
}, void 0, { Parent: Error }), ye = /* @__PURE__*/ ((e) => {
	let t = (n, r, i, a) => {
		let o = i ? {
			...i,
			async: !1
		} : { async: !1 }, c = n._zod.run({
			value: r,
			issues: []
		}, o);
		if (c instanceof Promise) throw new O();
		if (c.issues.length) {
			let n = new ((a?.Err) ?? e)(c.issues.map((e) => y(e, o, A())));
			throw s(n, a?.callee ?? t), n;
		}
		return c.value;
	};
	return t;
})(M), be = /* @__PURE__*/ ((e) => {
	let t = async (n, r, i, a) => {
		let o = i ? {
			...i,
			async: !0
		} : { async: !0 }, c = n._zod.run({
			value: r,
			issues: []
		}, o);
		if (c instanceof Promise && (c = await c), c.issues.length) {
			let n = new ((a?.Err) ?? e)(c.issues.map((e) => y(e, o, A())));
			throw s(n, a?.callee ?? t), n;
		}
		return c.value;
	};
	return t;
})(M), xe = /* @__PURE__*/ ((e) => (t, n, r) => {
	let i = r ? {
		...r,
		async: !1
	} : { async: !1 }, a = t._zod.run({
		value: n,
		issues: []
	}, i);
	if (a instanceof Promise) throw new O();
	return a.issues.length ? Se(e, a.issues, i) : {
		success: !0,
		data: a.value
	};
})(M);
function Se(e, t, n) {
	let r;
	return {
		success: !1,
		get error() {
			return r || (r = new e(t.map((e) => y(e, n, A()))), t = void 0, n = void 0), r;
		},
		set error(e) {
			r = e, t = void 0, n = void 0;
		}
	};
}
var Ce = /* @__PURE__*/ ((e) => async (t, n, r) => {
	let i = r ? {
		...r,
		async: !0
	} : { async: !0 }, a = t._zod.run({
		value: n,
		issues: []
	}, i);
	return a instanceof Promise && (a = await a), a.issues.length ? Se(e, a.issues, i) : {
		success: !0,
		data: a.value
	};
})(M), we = /^[\s\S]{0,}$/, Te = /^-?\d+(?:\.\d+)?$/, Ee = /^(?:true|false)$/i, N = /*@__PURE__*/ D("$ZodCheck", (e, t) => {
	var n;
	e._zod ?? (e._zod = {}), e._zod.def = t, (n = e._zod).onattach ?? (n.onattach = []);
}), De = (e) => {
	let t = e.value;
	return !i(t) && t.length !== void 0;
}, P = {
	number: "number",
	bigint: "bigint",
	object: "date"
}, Oe = /*@__PURE__*/ D("$ZodCheckLessThan", (e, t) => {
	N.init(e, t);
	let n = P[typeof t.value];
	e._zod.check = (r) => {
		(t.inclusive ? r.value <= t.value : r.value < t.value) || r.issues.push({
			origin: P[typeof r.value] ?? n,
			code: "too_big",
			maximum: typeof t.value == "object" ? t.value.getTime() : t.value,
			input: r.value,
			inclusive: t.inclusive,
			inst: e,
			continue: !t.abort
		});
	};
}), ke = /*@__PURE__*/ D("$ZodCheckGreaterThan", (e, t) => {
	N.init(e, t);
	let n = P[typeof t.value];
	e._zod.check = (r) => {
		(t.inclusive ? r.value >= t.value : r.value > t.value) || r.issues.push({
			origin: P[typeof r.value] ?? n,
			code: "too_small",
			minimum: typeof t.value == "object" ? t.value.getTime() : t.value,
			input: r.value,
			inclusive: t.inclusive,
			inst: e,
			continue: !t.abort
		});
	};
}), Ae = /*@__PURE__*/ D("$ZodCheckNumberFormat", (e, t) => {
	N.init(e, t), t.format = t.format || "float64";
	let n = t.format?.includes("int"), r = n ? "int" : "number", [i, a] = ee[t.format];
	e._zod.check = (o) => {
		let s = o.value;
		if (n) {
			if (!Number.isInteger(s)) {
				o.issues.push({
					expected: r,
					format: t.format,
					code: "invalid_type",
					continue: !1,
					input: s,
					inst: e
				});
				return;
			}
			if (!Number.isSafeInteger(s)) {
				s > 0 ? o.issues.push({
					input: s,
					code: "too_big",
					maximum: 2 ** 53 - 1,
					note: "Integers must be within the safe integer range.",
					inst: e,
					origin: r,
					inclusive: !0,
					continue: !t.abort
				}) : o.issues.push({
					input: s,
					code: "too_small",
					minimum: -(2 ** 53 - 1),
					note: "Integers must be within the safe integer range.",
					inst: e,
					origin: r,
					inclusive: !0,
					continue: !t.abort
				});
				return;
			}
		}
		s < i && o.issues.push({
			origin: "number",
			input: s,
			code: "too_small",
			minimum: i,
			inclusive: !0,
			inst: e,
			continue: !t.abort
		}), s > a && o.issues.push({
			origin: "number",
			input: s,
			code: "too_big",
			maximum: a,
			inclusive: !0,
			inst: e,
			continue: !t.abort
		});
	};
}), je = /*@__PURE__*/ D("$ZodCheckMaxLength", (e, t) => {
	var n;
	N.init(e, t), (n = e._zod.def).when ?? (n.when = De), e._zod.check = (n) => {
		let r = n.value, i = r.length;
		if ((typeof r == "string" && i > t.maximum ? b(r) : i) <= t.maximum) return;
		let a = ae(r);
		n.issues.push({
			origin: a,
			code: "too_big",
			maximum: t.maximum,
			inclusive: !0,
			input: r,
			inst: e,
			continue: !t.abort
		});
	};
}), Me = /*@__PURE__*/ D("$ZodCheckMinLength", (e, t) => {
	var n;
	N.init(e, t), (n = e._zod.def).when ?? (n.when = De), e._zod.check = (n) => {
		let r = n.value, i = r.length;
		if ((typeof r == "string" && i >= t.minimum && i < t.minimum * 2 ? b(r) : i) >= t.minimum) return;
		let a = ae(r);
		n.issues.push({
			origin: a,
			code: "too_small",
			minimum: t.minimum,
			inclusive: !0,
			input: r,
			inst: e,
			continue: !t.abort
		});
	};
}), Ne = {
	major: 4,
	minor: 6,
	patch: 7
}, F = /*@__PURE__*/ D("$ZodType", (e, t) => {
	var n;
	e ?? (e = {}), e._zod.def = t, e._zod.bag = e._zod.bag || {}, e._zod.version = Ne;
	let r = e._zod.def.checks, i = e._zod.traits.has("$ZodCheck") ? [e, ...r ?? []] : r?.length ? [...r] : [];
	for (let t of i) for (let n of t._zod.onattach) n(e);
	if (i.length === 0) (n = e._zod).deferred ?? (n.deferred = []), e._zod.deferred?.push(() => {
		e._zod.run = e._zod.parse;
	});
	else {
		let t = (t, n, r) => {
			if (t.memo) return t;
			let i = g(t), a;
			for (let o of n) {
				if (o._zod.def.when) {
					if (te(t) || !o._zod.def.when(t)) continue;
				} else if (i) continue;
				let n = t.issues.length, s = o._zod.check(t);
				if (s instanceof Promise && r?.async === !1) throw new O();
				if (a || s instanceof Promise) a = (a ?? Promise.resolve()).then(async () => {
					await s, t.issues.length !== n && (v(t.issues, n, e), i || (i = g(t, n)));
				});
				else {
					if (t.issues.length === n) continue;
					v(t.issues, n, e), i || (i = g(t, n));
				}
			}
			return a ? a.then(() => t) : t;
		}, n = (n, r, a) => {
			if (g(n)) return n.aborted = !0, n;
			let o = t(r, i, a);
			if (o instanceof Promise) {
				if (a.async === !1) throw new O();
				return o.then((t) => e._zod.parse(t, a));
			}
			return e._zod.parse(o, a);
		};
		e._zod.run = (r, a) => {
			if (a.skipChecks) return e._zod.parse(r, a);
			if (a.direction === "backward") {
				let t = e._zod.parse({
					value: r.value,
					issues: []
				}, {
					...a,
					skipChecks: !0
				});
				return t instanceof Promise ? t.then((e) => n(e, r, a)) : n(t, r, a);
			}
			let o = e._zod.parse(r, a);
			if (o instanceof Promise) {
				if (a.async === !1) throw new O();
				return o.then((e) => t(e, i, a));
			}
			return t(o, i, a);
		};
	}
}, {
	get "~standard"() {
		return ce(this, "~standard", Ie(this));
	},
	set "~standard"(e) {
		x(this, "~standard", e);
	}
}), Pe = (e, t) => e.issues.length ? { issues: e.issues.map((e) => y(e, t, A())) } : { value: e.value };
async function Fe(e, t) {
	let n = { async: !0 };
	return Pe(await e._zod.run({
		value: t,
		issues: []
	}, n), n);
}
function Ie(e) {
	return {
		validate: (t) => {
			let n = { async: !1 };
			try {
				let r = e._zod.run({
					value: t,
					issues: []
				}, n);
				if (!(r instanceof Promise)) return Pe(r, n);
			} catch {}
			return Fe(e, t);
		},
		vendor: "zod",
		version: 1
	};
}
var Le = /*@__PURE__*/ D("$ZodString", (e, t) => {
	F.init(e, t), e._zod.pattern = t.pattern ?? we, e._zod.parse = (n, r) => {
		if (t.coerce) try {
			n.value = String(n.value);
		} catch {}
		return typeof n.value == "string" || n.issues.push({
			expected: "string",
			code: "invalid_type",
			input: n.value,
			inst: e
		}), n;
	};
}), Re = /*@__PURE__*/ D("$ZodNumber", (e, t) => {
	F.init(e, t), e._zod.pattern = Te, e._zod.parse = (n, r) => {
		if (t.coerce) try {
			n.value = Number(n.value);
		} catch {}
		let i = n.value;
		if (typeof i == "number" && !Number.isNaN(i) && Number.isFinite(i)) return n;
		let a = typeof i == "number" ? Number.isNaN(i) ? "NaN" : Number.isFinite(i) ? void 0 : String(i) : void 0;
		return n.issues.push({
			expected: "number",
			code: "invalid_type",
			input: i,
			inst: e,
			...a ? { received: a } : {}
		}), n;
	};
}), ze = /*@__PURE__*/ D("$ZodNumberFormat", (e, t) => {
	Ae.init(e, t), Re.init(e, t);
}), Be = /*@__PURE__*/ D("$ZodBoolean", (e, t) => {
	F.init(e, t), e._zod.pattern = Ee, e._zod.parse = (n, r) => {
		if (t.coerce) try {
			n.value = !!n.value;
		} catch {}
		let i = n.value;
		return typeof i == "boolean" || n.issues.push({
			expected: "boolean",
			code: "invalid_type",
			input: i,
			inst: e
		}), n;
	};
}), Ve = /*@__PURE__*/ D("$ZodUnknown", (e, t) => {
	F.init(e, t), e._zod.parse = (e) => e;
});
function He(e, t) {
	let n = re(e._zod.def.checks);
	if (!n.length) return;
	let r = Math.min(...n.map((e) => e[0])), i = e._zod.parse;
	e._zod.parse = (n, a) => {
		let o = t(n.value);
		if (o > r && !a?.skipChecks && a?.direction !== "backward") {
			for (let [t, r] of re(e._zod.def.checks)) if (o > t && r._zod.def.when(n)) {
				let t = n.issues.length;
				return r._zod.check(n), n.issues[t].continue = !1, v(n.issues, t, e), n;
			}
		}
		return i(n, a);
	};
}
var Ue = (e) => Array.isArray(e) ? e.length : -1;
function We(e, t, n) {
	e.issues.length && t.issues.push(...ne(n, e.issues)), t.value[n] = e.value;
}
var Ge = /*@__PURE__*/ D("$ZodArray", (e, t) => {
	F.init(e, t);
	let n = k.memoizer;
	n?.attach(e), e._zod.parse = (r, i) => {
		let a = r.value;
		if (!Array.isArray(a)) return r.issues.push({
			expected: "array",
			code: "invalid_type",
			input: a,
			inst: e
		}), r;
		r.value = n ? n.alloc(e, r, Array(a.length), i) : Array(a.length);
		let o = [], s = i?.abortEarly;
		for (let e = 0; e < a.length; e++) {
			let n = a[e], c = t.element._zod.run({
				value: n,
				issues: []
			}, i);
			if (c instanceof Promise) o.push(c.then((t) => We(t, r, e)));
			else if (We(c, r, e), s && c.issues.length !== 0 && g(c)) break;
		}
		return o.length ? Promise.all(o).then(() => r) : r;
	}, He(e, Ue);
});
function I(e, t, n, r, i, a) {
	let o = n in r, s = a === "optional";
	if (o || !s || i !== "optional") {
		if (e.issues.length) {
			if (i !== void 0 && s && !o) return;
			t.issues.push(...ne(n, e.issues));
		}
		!o && i === void 0 ? e.issues.length || t.issues.push({
			code: "invalid_type",
			expected: "nonoptional",
			input: void 0,
			path: [n]
		}) : e.value === void 0 ? (o || i === "defaulted" && !s) && (t.value[n] = void 0) : t.value[n] = e.value;
	}
}
var Ke = [];
function qe(e) {
	let t = Object.keys(e.shape), n = Object.getOwnPropertySymbols(e.shape), r = n.length ? n : Ke, i = r.length ? [...t, ...r] : t;
	for (let t of i) if (!e.shape?.[t]?._zod?.traits?.has("$ZodType")) throw Error(`Invalid element at key "${String(t)}": expected a Zod schema`);
	let a = h(e.shape);
	return {
		...e,
		allKeys: i,
		symbolKeys: r,
		keySet: new Set(t),
		numKeys: t.length,
		optionalKeys: new Set(a)
	};
}
function Je(e, t, n, r, i, a, o) {
	let s = [], c = i.keySet, l = i.catchall._zod, u = l.def.type, d = l.optin, f = l.optout, p = 0;
	for (let i in t) {
		if (o && n.issues.length !== p) {
			if (g(n, p)) break;
			p = n.issues.length;
		}
		if (c.has(i)) continue;
		if (i === "__proto__") {
			u === "never" && s.push(i);
			continue;
		}
		if (u === "never") {
			s.push(i);
			continue;
		}
		let a = l.run({
			value: t[i],
			issues: []
		}, r);
		a instanceof Promise ? e.push(a.then((e) => I(e, n, i, t, d, f))) : I(a, n, i, t, d, f);
	}
	return s.length && n.issues.push({
		code: "unrecognized_keys",
		keys: s,
		input: t,
		inst: a,
		continue: !0
	}), e.length ? Promise.all(e).then(() => n) : n;
}
var Ye = /*@__PURE__*/ D("$ZodObject", (e, t) => {
	F.init(e, t);
	let n = Object.getOwnPropertyDescriptor(t, "shape"), i = n?.get ? n.get.raw : t.shape ?? {};
	if (i) {
		let e = () => {
			let n = { ...i };
			return Object.defineProperty(t, "shape", { value: n }), e.raw = n, n;
		};
		e.raw = i, Object.defineProperty(t, "shape", { get: e });
	}
	let a = r(() => qe(t));
	w(e, "propValues", (e) => {
		let t = e.def.shape, n = {};
		for (let e in t) {
			let r = t[e]._zod;
			if (r.values) {
				Object.prototype.hasOwnProperty.call(n, e) || o(n, e, /* @__PURE__ */ new Set());
				for (let t of r.values) n[e].add(t);
				r.optin !== void 0 && n[e].add(void 0);
			}
		}
		return n;
	});
	let s = c, l = t.catchall, u, d = k.memoizer;
	d?.attach(e), e._zod.parse = (t, n) => {
		u ?? (u = a.value);
		let r = t.value;
		if (!s(r)) return t.issues.push({
			expected: "object",
			code: "invalid_type",
			input: r,
			inst: e
		}), t;
		t.value = d ? d.alloc(e, t, {}, n) : {};
		let i = [], o = u.shape, c = n?.abortEarly, f = t.issues.length;
		for (let e of u.allKeys) {
			if (c && t.issues.length !== f) {
				if (g(t, f)) break;
				f = t.issues.length;
			}
			if (e === "__proto__") continue;
			let a = o[e], s = a._zod.optin, l = a._zod.optout, u = a._zod.run({
				value: r[e],
				issues: []
			}, n);
			u instanceof Promise ? i.push(u.then((n) => I(n, t, e, r, s, l))) : I(u, t, e, r, s, l);
		}
		return l ? Je(i, r, t, n, a.value, e, c === !0) : i.length ? Promise.all(i).then(() => t) : t;
	};
}), Xe = /*@__PURE__*/ D("$ZodEnum", (t, n) => {
	F.init(t, n);
	let r = e(n.entries), i = new Set(r);
	t._zod.values = i, w(t, "pattern", (t) => {
		let n = e(t.def.entries).filter((e) => d.has(typeof e));
		return RegExp(n.length ? `^(${n.map((e) => f(e.toString())).join("|")})$` : "^[^\\s\\S]$");
	}), t._zod.parse = (e, n) => {
		let a = e.value;
		return i.has(a) || e.issues.push({
			code: "invalid_value",
			values: r,
			input: a,
			inst: t
		}), e;
	};
}), Ze = /*@__PURE__*/ D("$ZodLiteral", (e, t) => {
	F.init(e, t);
	let n = new Set(t.values);
	e._zod.values = n, w(e, "pattern", (e) => {
		let t = e.def.values;
		return RegExp(t.length ? `^(${t.map((e) => typeof e == "string" ? f(e) : e ? f(e.toString()) : String(e)).join("|")})$` : "^[^\\s\\S]$");
	}), e._zod.parse = (r, i) => {
		let a = r.value;
		return n.has(a) || r.issues.push({
			code: "invalid_value",
			values: t.values,
			input: a,
			inst: e
		}), r;
	};
});
function Qe(e, t) {
	return e.value = t.issues.length ? void 0 : t.value, e;
}
var $e = /*@__PURE__*/ D("$ZodOptional", (e, t) => {
	F.init(e, t), w(e, "optin", (e) => e.def.innerType._zod.optin === "defaulted" ? "defaulted" : "optional"), e._zod.optout = "optional", w(e, "values", (e) => {
		let t = e.def.innerType._zod.values;
		return t ? /* @__PURE__ */ new Set([...t, void 0]) : void 0;
	}), w(e, "pattern", (e) => {
		let t = e.def.innerType._zod.pattern;
		return t ? RegExp(`^(${a(t.source)})?$`) : void 0;
	}), e._zod.parse = (e, n) => {
		if (e.value === void 0) {
			if (t.innerType._zod.optin !== "defaulted") return e;
			let r = t.innerType._zod.run({
				value: e.value,
				issues: []
			}, n);
			return r instanceof Promise ? r.then((t) => Qe(e, t)) : Qe(e, r);
		}
		return t.innerType._zod.run(e, n);
	};
}), et = /*@__PURE__*/ D("$ZodDefault", (e, t) => {
	F.init(e, t), e._zod.optin = "defaulted", w(e, "values", (e) => e.def.innerType._zod.values), e._zod.parse = (e, n) => {
		if (n.direction === "backward") return t.innerType._zod.run(e, n);
		if (e.value === void 0) return e.value = t.defaultValue, e;
		let r = t.innerType._zod.run(e, n);
		return r instanceof Promise ? r.then((e) => tt(e, t)) : tt(r, t);
	};
});
function tt(e, t) {
	return e.value === void 0 && (e.value = t.defaultValue), e;
}
var nt = /*@__PURE__*/ D("$ZodCustom", (e, t) => {
	N.init(e, t), F.init(e, t), e._zod.parse = (e, t) => e, e._zod.check = (n) => {
		let r = n.value, i = t.fn(r);
		if (i instanceof Promise) return i.then((t) => L(t, n, r, e));
		L(i, n, r, e);
	};
});
function L(e, t, n, r) {
	if (!e) {
		let e = {
			code: "custom",
			input: n,
			inst: r,
			path: [...r._zod.def.path ?? []],
			continue: !r._zod.def.abort
		};
		r._zod.def.params && (e.params = r._zod.def.params), t.issues.push(oe(e));
	}
}
// ../../node_modules/zod/v4/core/api.js
function R(e) {
	return e.checks && (e.checks = [...e.checks]), e;
}
// @__NO_SIDE_EFFECTS__
function rt(e, t) {
	return new e(R({
		type: "string",
		...m(t)
	}));
}
// @__NO_SIDE_EFFECTS__
function it(e, t) {
	return new e(R({
		type: "number",
		checks: [],
		...m(t)
	}));
}
// @__NO_SIDE_EFFECTS__
function at(e, t) {
	return new e({
		type: "number",
		check: "number_format",
		abort: !1,
		format: "safeint",
		...m(t)
	});
}
// @__NO_SIDE_EFFECTS__
function ot(e, t) {
	return new e({
		type: "boolean",
		...m(t)
	});
}
// @__NO_SIDE_EFFECTS__
function st(e) {
	return new e({ type: "unknown" });
}
// @__NO_SIDE_EFFECTS__
function z(e, t) {
	return new Oe({
		check: "less_than",
		...m(t),
		value: e,
		inclusive: !0
	});
}
// @__NO_SIDE_EFFECTS__
function ct(e, t) {
	return new ke({
		check: "greater_than",
		...m(t),
		value: e,
		inclusive: !1
	});
}
// @__NO_SIDE_EFFECTS__
function B(e, t) {
	return new ke({
		check: "greater_than",
		...m(t),
		value: e,
		inclusive: !0
	});
}
// @__NO_SIDE_EFFECTS__
function lt(e) {
	return /* @__PURE__ */ ct(0, e);
}
// @__NO_SIDE_EFFECTS__
function V(e, t) {
	return new je({
		check: "max_length",
		...m(t),
		maximum: e
	});
}
// @__NO_SIDE_EFFECTS__
function H(e, t) {
	return new Me({
		check: "min_length",
		...m(t),
		minimum: e
	});
}
// @__NO_SIDE_EFFECTS__
function ut(e, t, n) {
	return new e({
		type: "custom",
		check: "custom",
		fn: t,
		...m(n)
	});
}
// ../../node_modules/zod/v4/mini/schemas.js
var U = /*@__PURE__*/ D("ZodMiniType", (e, t) => {
	if (!e._zod) throw Error("Uninitialized schema in ZodMiniType.");
	F.init(e, t), e.def = t, e.type = t.type;
}, {
	get with() {
		return this.check;
	},
	set with(e) {
		x(this, "with", e);
	},
	parse(e, t) {
		return ye(this, e, t, { callee: this.parse });
	},
	parseAsync(e, t) {
		return be(this, e, t, { callee: this.parseAsync });
	},
	safeParse(e, t) {
		return xe(this, e, t);
	},
	safeParseAsync(e, t) {
		return Ce(this, e, t);
	},
	check(...e) {
		let t = this.def;
		return this.clone({
			...t,
			checks: [...t.checks ?? [], ...e.map((e) => typeof e == "function" ? { _zod: {
				check: e,
				def: { check: "custom" },
				onattach: []
			} } : e)]
		}, { parent: !0 });
	},
	clone(e, t) {
		return p(this, e, t);
	},
	brand() {
		return this;
	},
	register(e, t) {
		return e.add(this, t), this;
	},
	apply(e, ...t) {
		return t.length === 0 ? e(this) : e(this, ...t);
	}
}), dt = /*@__PURE__*/ D("ZodMiniString", (e, t) => {
	Le.init(e, t), U.init(e, t);
});
// @__NO_SIDE_EFFECTS__
function W(e) {
	return /* @__PURE__ */ rt(dt, e);
}
var ft = /*@__PURE__*/ D("ZodMiniNumber", (e, t) => {
	Re.init(e, t), U.init(e, t);
});
// @__NO_SIDE_EFFECTS__
function pt(e) {
	return /* @__PURE__ */ it(ft, e);
}
var mt = /*@__PURE__*/ D("ZodMiniNumberFormat", (e, t) => {
	ze.init(e, t), ft.init(e, t);
});
function G(e) {
	return /* @__PURE__ */ at(mt, e);
}
var ht = /*@__PURE__*/ D("ZodMiniBoolean", (e, t) => {
	Be.init(e, t), U.init(e, t);
});
// @__NO_SIDE_EFFECTS__
function K(e) {
	return /* @__PURE__ */ ot(ht, e);
}
var gt = /*@__PURE__*/ D("ZodMiniUnknown", (e, t) => {
	Ve.init(e, t), U.init(e, t);
});
// @__NO_SIDE_EFFECTS__
function _t() {
	return /* @__PURE__ */ st(gt);
}
var vt = /*@__PURE__*/ D("ZodMiniArray", (e, t) => {
	Ge.init(e, t), U.init(e, t);
});
// @__NO_SIDE_EFFECTS__
function yt(e, t) {
	return new vt({
		type: "array",
		element: e,
		...m(t)
	});
}
var bt = /*@__PURE__*/ D("ZodMiniObject", (e, t) => {
	Ye.init(e, t), U.init(e, t), fe(e, "shape", (e) => e._zod.def.shape, !1);
});
// @__NO_SIDE_EFFECTS__
function q(e, t) {
	return new bt({
		type: "object",
		shape: e ?? {},
		...m(t)
	});
}
var xt = /*@__PURE__*/ D("ZodMiniEnum", (e, t) => {
	Xe.init(e, t), U.init(e, t), e.options = [...e._zod.values];
});
// @__NO_SIDE_EFFECTS__
function St(e, t) {
	return new xt({
		type: "enum",
		entries: Array.isArray(e) ? Object.fromEntries(e.map((e) => [e, e])) : e,
		...m(t)
	});
}
var Ct = /*@__PURE__*/ D("ZodMiniLiteral", (e, t) => {
	Ze.init(e, t), U.init(e, t);
});
// @__NO_SIDE_EFFECTS__
function J(e, t) {
	return new Ct({
		type: "literal",
		values: Array.isArray(e) ? e : [e],
		...m(t)
	});
}
var wt = /*@__PURE__*/ D("ZodMiniOptional", (e, t) => {
	$e.init(e, t), U.init(e, t);
});
// @__NO_SIDE_EFFECTS__
function Tt(e) {
	return new wt({
		type: "optional",
		innerType: e
	});
}
var Et = /*@__PURE__*/ D("ZodMiniDefault", (e, t) => {
	et.init(e, t), U.init(e, t);
});
// @__NO_SIDE_EFFECTS__
function Dt(e, t) {
	return new Et({
		type: "default",
		innerType: e,
		get defaultValue() {
			return typeof t == "function" ? t() : u(t);
		}
	});
}
var Ot = /*@__PURE__*/ D("ZodMiniCustom", (e, t) => {
	nt.init(e, t), U.init(e, t);
});
// @__NO_SIDE_EFFECTS__
function Y(e, t = {}) {
	return /* @__PURE__ */ ut(Ot, e, t);
}
var X = (/* @__PURE__ */ W()).check(/* @__PURE__ */ H(1), /* @__PURE__ */ V(128)), kt = (/* @__PURE__ */ W()).check(/* @__PURE__ */ V(64)), Z = (e, t) => (/* @__PURE__ */ pt()).check(/* @__PURE__ */ B(e), /* @__PURE__ */ z(t)), At = /* @__PURE__ */ q({
	id: X,
	name: kt
}), jt = /* @__PURE__ */ St([
	"open",
	"host_only",
	"turn",
	"allowlist"
]);
(/* @__PURE__ */ W()).check(/* @__PURE__ */ V(32)), Z(0, 1), G().check(/* @__PURE__ */ B(0), /* @__PURE__ */ z(1e3)), Z(0, 86400);
var Mt = /* @__PURE__ */ q({
	top: Z(0, 1e4),
	right: Z(0, 1e4),
	bottom: Z(0, 1e4),
	left: Z(0, 1e4)
}), Nt = {
	"sdk.init": /* @__PURE__ */ q({
		protocolVersion: /* @__PURE__ */ J(1),
		launchCode: (/* @__PURE__ */ W()).check(/* @__PURE__ */ H(16), /* @__PURE__ */ V(256)),
		player: /* @__PURE__ */ q({
			name: kt,
			locale: (/* @__PURE__ */ W()).check(/* @__PURE__ */ V(16))
		}),
		room: /* @__PURE__ */ q({ id: X })
	}),
	"app.pause": /* @__PURE__ */ q({}),
	"app.resume": /* @__PURE__ */ q({}),
	"app.close": /* @__PURE__ */ q({}),
	"audio.mute": /* @__PURE__ */ q({ muted: /* @__PURE__ */ K() }),
	"room.members": /* @__PURE__ */ q({ members: (/* @__PURE__ */ yt(At)).check(/* @__PURE__ */ V(64)) }),
	"ui.insets": Mt,
	"voice.state": /* @__PURE__ */ q({
		connected: /* @__PURE__ */ K(),
		canSpeak: /* @__PURE__ */ K(),
		mode: jt
	}),
	"voice.speaking": /* @__PURE__ */ q({
		playerId: X,
		level: Z(0, 1)
	})
}, Pt = /* @__PURE__ */ q({
	pt: /* @__PURE__ */ J("playtogether"),
	v: /* @__PURE__ */ J(1),
	type: (/* @__PURE__ */ W()).check(/* @__PURE__ */ V(64)),
	id: (/* @__PURE__ */ W()).check(/* @__PURE__ */ H(1), /* @__PURE__ */ V(64)),
	instance: /* @__PURE__ */ Tt((/* @__PURE__ */ W()).check(/* @__PURE__ */ V(64))),
	nonce: /* @__PURE__ */ Tt((/* @__PURE__ */ W()).check(/* @__PURE__ */ V(128))),
	payload: /* @__PURE__ */ _t()
});
function Ft(e) {
	try {
		return new TextEncoder().encode(JSON.stringify(e)).length;
	} catch {
		return Infinity;
	}
}
function It(e) {
	if (Ft(e) > 8192) return {
		ok: !1,
		error: "too_large"
	};
	let t = Pt.safeParse(e);
	return t.success ? {
		ok: !0,
		value: t.data
	} : {
		ok: !1,
		error: "bad_envelope"
	};
}
function Lt(e) {
	let t = It(e);
	if (!t.ok) return t;
	let n = Nt[t.value.type];
	if (!n) return {
		ok: !1,
		error: "unknown_type"
	};
	let r = n.safeParse(t.value.payload);
	return r.success ? {
		ok: !0,
		value: {
			...t.value,
			payload: r.data
		}
	} : {
		ok: !1,
		error: "bad_payload"
	};
}
(/* @__PURE__ */ q({
	roomId: X,
	gameInstanceId: X,
	sequence: G().check(/* @__PURE__ */ B(0)),
	mode: jt,
	allowedSpeakerIds: /* @__PURE__ */ Dt((/* @__PURE__ */ yt(X)).check(/* @__PURE__ */ V(64)), []),
	effectiveAtServerMs: G().check(/* @__PURE__ */ lt()),
	expiresAtServerMs: G().check(/* @__PURE__ */ lt())
})).check(/* @__PURE__ */ Y((e) => e.expiresAtServerMs > e.effectiveAtServerMs, "expiresAtServerMs 必須晚於 effectiveAtServerMs"), /* @__PURE__ */ Y((e) => e.mode !== "turn" || e.allowedSpeakerIds.length === 1, "turn 模式必須剛好一位發言者"), /* @__PURE__ */ Y((e) => e.mode !== "allowlist" || e.allowedSpeakerIds.length >= 1, "allowlist 模式至少要一位發言者"), /* @__PURE__ */ Y((e) => e.mode !== "open" && e.mode !== "host_only" || e.allowedSpeakerIds.length === 0, "open / host_only 模式不帶發言者名單"), /* @__PURE__ */ Y((e) => new Set(e.allowedSpeakerIds).size === e.allowedSpeakerIds.length, "發言者名單不得重複")), (/* @__PURE__ */ W()).check(/* @__PURE__ */ H(16), /* @__PURE__ */ V(256)), G().check(/* @__PURE__ */ B(0));
// \0@oxc-project+runtime@0.156.0/helpers/esm/typeof.js
function Q(e) {
	"@babel/helpers - typeof";
	return Q = typeof Symbol == "function" && typeof Symbol.iterator == "symbol" ? function(e) {
		return typeof e;
	} : function(e) {
		return e && typeof Symbol == "function" && e.constructor === Symbol && e !== Symbol.prototype ? "symbol" : typeof e;
	}, Q(e);
}
// \0@oxc-project+runtime@0.156.0/helpers/esm/toPrimitive.js
function Rt(e, t) {
	if (Q(e) != "object" || !e) return e;
	var n = e[Symbol.toPrimitive];
	if (n !== void 0) {
		var r = n.call(e, t || "default");
		if (Q(r) != "object") return r;
		throw TypeError("@@toPrimitive must return a primitive value.");
	}
	return (t === "string" ? String : Number)(e);
}
// \0@oxc-project+runtime@0.156.0/helpers/esm/toPropertyKey.js
function zt(e) {
	var t = Rt(e, "string");
	return Q(t) == "symbol" ? t : t + "";
}
// \0@oxc-project+runtime@0.156.0/helpers/esm/defineProperty.js
function Bt(e, t, n) {
	return (t = zt(t)) in e ? Object.defineProperty(e, t, {
		value: n,
		enumerable: !0,
		configurable: !0,
		writable: !0
	}) : e[t] = n, e;
}
// src/index.ts
var Vt = "0.1.0";
function Ht(e = window) {
	return {
		embedded: e.parent !== e,
		parent: e.parent,
		post: (t, n) => e.parent.postMessage(t, n),
		listen(t) {
			let n = (e) => t({
				origin: e.origin,
				source: e.source,
				data: e.data
			});
			return e.addEventListener("message", n), () => e.removeEventListener("message", n);
		}
	};
}
var $ = class {
	constructor() {
		Bt(this, "map", /* @__PURE__ */ new Map());
	}
	on(e, t) {
		let n = this.map.get(e);
		return n || this.map.set(e, n = /* @__PURE__ */ new Set()), n.add(t), () => n.delete(t);
	}
	emit(e, t) {
		for (let n of this.map.get(e) ?? []) try {
			n(t);
		} catch (e) {
			console.error("[playtogether] listener error", e);
		}
	}
}, Ut = class extends Error {
	constructor() {
		super("無法和玩聚平台完成握手（逾時）"), this.name = "HandshakeTimeoutError";
	}
};
function Wt() {
	let e = /* @__PURE__ */ new Uint8Array(16);
	return crypto.getRandomValues(e), Array.from(e, (e) => e.toString(16).padStart(2, "0")).join("");
}
function Gt(e, t, n, r) {
	let i = new $(), a = new $(), o = new $();
	return {
		client: {
			mode: e,
			player: { ...t.player },
			launch: { code: t.launchCode },
			room: { ...t.room },
			ready: () => n("sdk.ready", {}),
			loading: { progress: (e) => n("loading.progress", { value: Math.min(1, Math.max(0, e)) }) },
			game: {
				started: (e = {}) => n("game.started", e),
				ended: (e = {}) => n("game.ended", e)
			},
			ui: {
				share: () => n("ui.share", {}),
				fullscreen: () => n("ui.fullscreen", {}),
				orientation: (e) => n("ui.orientation", { orientation: e }),
				pttButton: ({ visible: e }) => n("ui.pttButton", { visible: e }),
				on: (e, t) => a.on(e, t)
			},
			voice: {
				pushToTalk: (e) => n("voice.pushToTalk", { action: e }),
				request: () => n("voice.request", { kind: "speak" }),
				on: (e, t) => o.on(e, t)
			},
			on: (e, t) => i.on(e, t),
			destroy: r
		},
		dispatch: (e) => {
			switch (e.type) {
				case "app.pause": return i.emit("pause", {});
				case "app.resume": return i.emit("resume", {});
				case "app.close": return i.emit("close", {});
				case "audio.mute": return i.emit("mute", e.payload);
				case "room.members": return i.emit("members", e.payload);
				case "ui.insets": return a.emit("insets", e.payload);
				case "voice.state": return o.emit("state", e.payload);
				case "voice.speaking": return o.emit("speaking", e.payload);
				default: return;
			}
		}
	};
}
function Kt() {
	return Gt("mock", {
		protocolVersion: 1,
		launchCode: `mock-${Wt()}`,
		player: {
			name: "測試玩家",
			locale: "zh-TW"
		},
		room: { id: "mock-room" }
	}, () => {}, () => {}).client;
}
function qt(e) {
	let t = e.transport ?? Ht();
	if (!t.embedded) return Promise.resolve(Kt());
	let n = e.platformOrigins.filter((e) => e !== "*");
	if (n.length === 0) return Promise.reject(/* @__PURE__ */ Error("platformOrigins 未設定：請填入玩聚平台的完整來源"));
	let r = e.timeoutMs ?? 5e3, i = e.helloIntervalMs ?? 250;
	return new Promise((e, a) => {
		let o = 0, s = null, c = "", l = "", u = null, d = (e, t, n) => (o += 1, {
			pt: "playtogether",
			v: 1,
			type: e,
			id: `g${o}`,
			...n ? {
				instance: c,
				nonce: l
			} : {},
			payload: t
		}), f = () => {
			let e = d("sdk.hello", { sdkVersion: Vt }, !1);
			for (let r of n) t.post(e, r);
		}, p = () => {
			clearInterval(h), clearTimeout(ee);
		}, m = t.listen((r) => {
			if (r.source !== t.parent || (s ? r.origin !== s : !n.includes(r.origin))) return;
			let i = Lt(r.data);
			if (!i.ok) return;
			let a = i.value;
			if (s) a.instance === c && a.nonce === l && u?.(a);
			else {
				if (a.type !== "sdk.init" || !a.instance || !a.nonce) return;
				s = r.origin, c = a.instance, l = a.nonce, p();
				let n = Gt("platform", a.payload, (e, n) => {
					t.post(d(e, n, !0), s);
				}, m);
				u = n.dispatch, e(n.client);
			}
		}), h = setInterval(f, i), ee = setTimeout(() => {
			p(), m(), a(new Ut());
		}, r);
		f();
	});
}
var Jt = { init: qt };
export { Ut as HandshakeTimeoutError, Jt as PlayTogether, Vt as SDK_VERSION, Ht as browserTransport, qt as init };
