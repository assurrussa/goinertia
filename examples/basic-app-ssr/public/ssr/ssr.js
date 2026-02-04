import { mergeProps, unref, withCtx, createTextVNode, useSSRContext, createVNode, openBlock, createBlock, Fragment, renderList, toDisplayString, reactive, ref, createSSRApp, h } from "vue";
import { ssrRenderAttrs, ssrInterpolate, ssrRenderComponent, ssrRenderList, ssrRenderAttr } from "vue/server-renderer";
import { Link, Deferred, createInertiaApp } from "@inertiajs/vue3";
import createServer from "@inertiajs/vue3/server";
import { renderToString } from "@vue/server-renderer";
const _export_sfc = (sfc, props) => {
  const target = sfc.__vccOpts || sfc;
  for (const [key, val] of props) {
    target[key] = val;
  }
  return target;
};
const _sfc_main$3 = {
  __name: "Error",
  __ssrInlineRender: true,
  props: {
    status: Number,
    message: String
  },
  setup(__props) {
    return (_ctx, _push, _parent, _attrs) => {
      _push(`<div${ssrRenderAttrs(mergeProps({ class: "error-page" }, _attrs))} data-v-a8366653><div class="error-code" data-v-a8366653>${ssrInterpolate(__props.status)}</div><p class="error-message" data-v-a8366653>${ssrInterpolate(__props.message || "Something went wrong")}</p>`);
      _push(ssrRenderComponent(unref(Link), {
        href: "/",
        class: "btn"
      }, {
        default: withCtx((_, _push2, _parent2, _scopeId) => {
          if (_push2) {
            _push2(` Go Home `);
          } else {
            return [
              createTextVNode(" Go Home ")
            ];
          }
        }),
        _: 1
      }, _parent));
      _push(`</div>`);
    };
  }
};
const _sfc_setup$3 = _sfc_main$3.setup;
_sfc_main$3.setup = (props, ctx) => {
  const ssrContext = useSSRContext();
  (ssrContext.modules || (ssrContext.modules = /* @__PURE__ */ new Set())).add("src/Pages/Error.vue");
  return _sfc_setup$3 ? _sfc_setup$3(props, ctx) : void 0;
};
const Error = /* @__PURE__ */ _export_sfc(_sfc_main$3, [["__scopeId", "data-v-a8366653"]]);
const __vite_glob_0_0 = /* @__PURE__ */ Object.freeze(/* @__PURE__ */ Object.defineProperty({
  __proto__: null,
  default: Error
}, Symbol.toStringTag, { value: "Module" }));
const _sfc_main$2 = {
  __name: "Home",
  __ssrInlineRender: true,
  props: {
    title: String,
    plan: String,
    heavy: Array
  },
  setup(__props) {
    return (_ctx, _push, _parent, _attrs) => {
      _push(`<div${ssrRenderAttrs(_attrs)} data-v-1086f386><h1 data-v-1086f386>${ssrInterpolate(__props.title)}</h1><p data-v-1086f386>Welcome to the Home page.</p><a href="/undefined-page" data-v-1086f386>Example link to a non-existent page</a><div class="panel" data-v-1086f386><h2 data-v-1086f386>Once prop</h2><p class="muted" data-v-1086f386>Plan: <strong data-v-1086f386>${ssrInterpolate(__props.plan)}</strong></p></div><div class="panel" data-v-1086f386><h2 data-v-1086f386>Deferred prop</h2>`);
      _push(ssrRenderComponent(unref(Deferred), { data: "heavy" }, {
        fallback: withCtx((_, _push2, _parent2, _scopeId) => {
          if (_push2) {
            _push2(`<div class="muted" data-v-1086f386${_scopeId}>Loading heavy data…</div>`);
          } else {
            return [
              createVNode("div", { class: "muted" }, "Loading heavy data…")
            ];
          }
        }),
        default: withCtx((_, _push2, _parent2, _scopeId) => {
          if (_push2) {
            _push2(`<ul class="list" data-v-1086f386${_scopeId}><!--[-->`);
            ssrRenderList(__props.heavy, (item) => {
              _push2(`<li data-v-1086f386${_scopeId}>${ssrInterpolate(item)}</li>`);
            });
            _push2(`<!--]--></ul>`);
          } else {
            return [
              createVNode("ul", { class: "list" }, [
                (openBlock(true), createBlock(Fragment, null, renderList(__props.heavy, (item) => {
                  return openBlock(), createBlock("li", { key: item }, toDisplayString(item), 1);
                }), 128))
              ])
            ];
          }
        }),
        _: 1
      }, _parent));
      _push(`</div></div>`);
    };
  }
};
const _sfc_setup$2 = _sfc_main$2.setup;
_sfc_main$2.setup = (props, ctx) => {
  const ssrContext = useSSRContext();
  (ssrContext.modules || (ssrContext.modules = /* @__PURE__ */ new Set())).add("src/Pages/Home.vue");
  return _sfc_setup$2 ? _sfc_setup$2(props, ctx) : void 0;
};
const Home = /* @__PURE__ */ _export_sfc(_sfc_main$2, [["__scopeId", "data-v-1086f386"]]);
const __vite_glob_0_1 = /* @__PURE__ */ Object.freeze(/* @__PURE__ */ Object.defineProperty({
  __proto__: null,
  default: Home
}, Symbol.toStringTag, { value: "Module" }));
const _sfc_main$1 = {
  __name: "Settings",
  __ssrInlineRender: true,
  props: {
    title: String,
    diagnostics: Object
  },
  setup(__props) {
    const form = reactive({
      name: "",
      email: ""
    });
    const errors = ref({});
    const status = ref("");
    return (_ctx, _push, _parent, _attrs) => {
      _push(`<div${ssrRenderAttrs(_attrs)} data-v-acb3112a><h1 data-v-acb3112a>${ssrInterpolate(__props.title)}</h1><p data-v-acb3112a>Settings page content.</p><div class="panel" data-v-acb3112a><h2 data-v-acb3112a>Optional prop</h2><p class="muted" data-v-acb3112a>Diagnostics are loaded only when requested.</p>`);
      if (__props.diagnostics) {
        _push(`<div class="diagnostics" data-v-acb3112a><pre data-v-acb3112a>${ssrInterpolate(__props.diagnostics)}</pre></div>`);
      } else {
        _push(ssrRenderComponent(unref(Link), {
          href: "/settings",
          class: "btn",
          "preserve-scroll": "",
          only: ["diagnostics"]
        }, {
          default: withCtx((_, _push2, _parent2, _scopeId) => {
            if (_push2) {
              _push2(` Load diagnostics `);
            } else {
              return [
                createTextVNode(" Load diagnostics ")
              ];
            }
          }),
          _: 1
        }, _parent));
      }
      _push(`</div><div class="panel" data-v-acb3112a><h2 data-v-acb3112a>Precognition form</h2><p class="muted" data-v-acb3112a>Validation-only request using Precognition headers.</p><form class="form-grid" data-v-acb3112a><label class="field" data-v-acb3112a> Name <input${ssrRenderAttr("value", form.name)} type="text" class="input" data-v-acb3112a>`);
      if (errors.value.name) {
        _push(`<span class="error" data-v-acb3112a>${ssrInterpolate(errors.value.name[0])}</span>`);
      } else {
        _push(`<!---->`);
      }
      _push(`</label><label class="field" data-v-acb3112a> Email <input${ssrRenderAttr("value", form.email)} type="email" class="input" data-v-acb3112a>`);
      if (errors.value.email) {
        _push(`<span class="error" data-v-acb3112a>${ssrInterpolate(errors.value.email[0])}</span>`);
      } else {
        _push(`<!---->`);
      }
      _push(`</label><div class="actions" data-v-acb3112a><button type="submit" class="btn" data-v-acb3112a>Validate only</button>`);
      if (status.value) {
        _push(`<span class="status" data-v-acb3112a>${ssrInterpolate(status.value)}</span>`);
      } else {
        _push(`<!---->`);
      }
      _push(`</div></form></div></div>`);
    };
  }
};
const _sfc_setup$1 = _sfc_main$1.setup;
_sfc_main$1.setup = (props, ctx) => {
  const ssrContext = useSSRContext();
  (ssrContext.modules || (ssrContext.modules = /* @__PURE__ */ new Set())).add("src/Pages/Settings.vue");
  return _sfc_setup$1 ? _sfc_setup$1(props, ctx) : void 0;
};
const Settings = /* @__PURE__ */ _export_sfc(_sfc_main$1, [["__scopeId", "data-v-acb3112a"]]);
const __vite_glob_0_2 = /* @__PURE__ */ Object.freeze(/* @__PURE__ */ Object.defineProperty({
  __proto__: null,
  default: Settings
}, Symbol.toStringTag, { value: "Module" }));
const _sfc_main = {
  __name: "Users",
  __ssrInlineRender: true,
  props: {
    title: String,
    users: Array,
    sort: String,
    page: Number,
    totalPages: Number,
    prevPage: [Number, null],
    nextPage: [Number, null]
  },
  setup(__props) {
    const sortOptions = [
      { value: "name", label: "Name ↑" },
      { value: "name_desc", label: "Name ↓" },
      { value: "id_desc", label: "ID ↓" },
      { value: "role", label: "Role" }
    ];
    return (_ctx, _push, _parent, _attrs) => {
      _push(`<div${ssrRenderAttrs(_attrs)} data-v-4a3ccb96><h1 data-v-4a3ccb96>${ssrInterpolate(__props.title)}</h1><div class="controls" data-v-4a3ccb96><span class="label" data-v-4a3ccb96>Sort:</span><!--[-->`);
      ssrRenderList(sortOptions, (option) => {
        _push(ssrRenderComponent(unref(Link), {
          key: option.value,
          href: `/users?sort=${option.value}`,
          class: ["chip", { active: __props.sort === option.value }]
        }, {
          default: withCtx((_, _push2, _parent2, _scopeId) => {
            if (_push2) {
              _push2(`${ssrInterpolate(option.label)}`);
            } else {
              return [
                createTextVNode(toDisplayString(option.label), 1)
              ];
            }
          }),
          _: 2
        }, _parent));
      });
      _push(`<!--]--></div><ul class="user-list" data-v-4a3ccb96><!--[-->`);
      ssrRenderList(__props.users, (user) => {
        _push(`<li class="user-item" data-v-4a3ccb96><div class="user-name" data-v-4a3ccb96>${ssrInterpolate(user.name)}</div><div class="user-meta" data-v-4a3ccb96>#${ssrInterpolate(user.id)} · ${ssrInterpolate(user.role)}</div></li>`);
      });
      _push(`<!--]--></ul><div class="pager" data-v-4a3ccb96>`);
      if (__props.prevPage) {
        _push(ssrRenderComponent(unref(Link), {
          href: `/users?sort=${__props.sort}&page=${__props.prevPage}`,
          class: "btn secondary"
        }, {
          default: withCtx((_, _push2, _parent2, _scopeId) => {
            if (_push2) {
              _push2(` Prev `);
            } else {
              return [
                createTextVNode(" Prev ")
              ];
            }
          }),
          _: 1
        }, _parent));
      } else {
        _push(`<!---->`);
      }
      if (__props.nextPage) {
        _push(ssrRenderComponent(unref(Link), {
          href: `/users?sort=${__props.sort}&page=${__props.nextPage}`,
          class: "btn",
          "preserve-scroll": "",
          only: ["users", "page", "prevPage", "nextPage", "totalPages"]
        }, {
          default: withCtx((_, _push2, _parent2, _scopeId) => {
            if (_push2) {
              _push2(` Load more `);
            } else {
              return [
                createTextVNode(" Load more ")
              ];
            }
          }),
          _: 1
        }, _parent));
      } else {
        _push(`<!---->`);
      }
      _push(`</div><p class="muted" data-v-4a3ccb96>Page ${ssrInterpolate(__props.page)} of ${ssrInterpolate(__props.totalPages)}</p></div>`);
    };
  }
};
const _sfc_setup = _sfc_main.setup;
_sfc_main.setup = (props, ctx) => {
  const ssrContext = useSSRContext();
  (ssrContext.modules || (ssrContext.modules = /* @__PURE__ */ new Set())).add("src/Pages/Users.vue");
  return _sfc_setup ? _sfc_setup(props, ctx) : void 0;
};
const Users = /* @__PURE__ */ _export_sfc(_sfc_main, [["__scopeId", "data-v-4a3ccb96"]]);
const __vite_glob_0_3 = /* @__PURE__ */ Object.freeze(/* @__PURE__ */ Object.defineProperty({
  __proto__: null,
  default: Users
}, Symbol.toStringTag, { value: "Module" }));
createServer((page) => createInertiaApp({
  page,
  render: renderToString,
  resolve: (name) => {
    const pages = /* @__PURE__ */ Object.assign({ "./Pages/Error.vue": __vite_glob_0_0, "./Pages/Home.vue": __vite_glob_0_1, "./Pages/Settings.vue": __vite_glob_0_2, "./Pages/Users.vue": __vite_glob_0_3 });
    return pages[`./Pages/${name}.vue`];
  },
  setup({ App, props, plugin }) {
    return createSSRApp({ render: () => h(App, props) }).use(plugin);
  }
}));
