import { jsxs, jsx } from "react/jsx-runtime";
import { Link, Deferred, usePage, createInertiaApp } from "@inertiajs/react";
import createServer from "@inertiajs/react/server";
import { renderToString } from "react-dom/server";
import axios from "axios";
import { useState } from "react";
function ErrorPage({ status, message }) {
  return /* @__PURE__ */ jsxs("div", { className: "error-page", children: [
    /* @__PURE__ */ jsx("div", { className: "error-code", children: status }),
    /* @__PURE__ */ jsx("p", { className: "error-message", children: message || "Something went wrong" }),
    /* @__PURE__ */ jsx(Link, { href: "/", className: "btn", children: "Go Home" })
  ] });
}
const __vite_glob_0_0 = /* @__PURE__ */ Object.freeze(/* @__PURE__ */ Object.defineProperty({
  __proto__: null,
  default: ErrorPage
}, Symbol.toStringTag, { value: "Module" }));
function Home({ title, plan, heavy = [] }) {
  return /* @__PURE__ */ jsxs("div", { children: [
    /* @__PURE__ */ jsx("h1", { children: title }),
    /* @__PURE__ */ jsx("p", { children: "Welcome to the Home page. This SSR version uses a React client." }),
    /* @__PURE__ */ jsx("a", { href: "/undefined-page", children: "Example link to a non-existent page" }),
    /* @__PURE__ */ jsxs("div", { className: "panel", children: [
      /* @__PURE__ */ jsx("h2", { children: "Once prop" }),
      /* @__PURE__ */ jsxs("p", { className: "muted", children: [
        "Plan: ",
        /* @__PURE__ */ jsx("strong", { children: plan })
      ] })
    ] }),
    /* @__PURE__ */ jsxs("div", { className: "panel", children: [
      /* @__PURE__ */ jsx("h2", { children: "Deferred prop" }),
      /* @__PURE__ */ jsx(Deferred, { data: "heavy", fallback: /* @__PURE__ */ jsx("div", { className: "muted", children: "Loading heavy data..." }), children: /* @__PURE__ */ jsx("ul", { className: "list", children: heavy.map((item) => /* @__PURE__ */ jsx("li", { children: item }, item)) }) })
    ] })
  ] });
}
const __vite_glob_0_1 = /* @__PURE__ */ Object.freeze(/* @__PURE__ */ Object.defineProperty({
  __proto__: null,
  default: Home
}, Symbol.toStringTag, { value: "Module" }));
function Settings({ title, diagnostics }) {
  const [form, setForm] = useState({ name: "", email: "" });
  const [errors, setErrors] = useState({});
  const [status, setStatus] = useState("");
  const handleChange = (event) => {
    const { name, value } = event.target;
    setForm((current) => ({ ...current, [name]: value }));
  };
  const validateForm = async (event) => {
    var _a, _b;
    event.preventDefault();
    setStatus("Validating...");
    setErrors({});
    const payload = new URLSearchParams();
    payload.set("name", form.name);
    payload.set("email", form.email);
    try {
      await axios.post("/users/create", payload, {
        headers: {
          Precognition: "true",
          "Precognition-Validate-Only": "name,email",
          "Content-Type": "application/x-www-form-urlencoded"
        }
      });
      setStatus("Looks good");
    } catch (error) {
      if (((_a = error.response) == null ? void 0 : _a.status) === 422) {
        setErrors(((_b = error.response.data) == null ? void 0 : _b.errors) ?? {});
        setStatus("Fix the errors above");
        return;
      }
      setStatus("Request failed");
    }
  };
  return /* @__PURE__ */ jsxs("div", { children: [
    /* @__PURE__ */ jsx("h1", { children: title }),
    /* @__PURE__ */ jsx("p", { children: "Settings page content." }),
    /* @__PURE__ */ jsxs("div", { className: "panel", children: [
      /* @__PURE__ */ jsx("h2", { children: "Optional prop" }),
      /* @__PURE__ */ jsx("p", { className: "muted", children: "Diagnostics are loaded only when requested." }),
      diagnostics ? /* @__PURE__ */ jsx("div", { className: "diagnostics", children: /* @__PURE__ */ jsx("pre", { children: JSON.stringify(diagnostics, null, 2) }) }) : /* @__PURE__ */ jsx(Link, { href: "/settings", className: "btn", preserveScroll: true, only: ["diagnostics"], children: "Load diagnostics" })
    ] }),
    /* @__PURE__ */ jsxs("div", { className: "panel", children: [
      /* @__PURE__ */ jsx("h2", { children: "Precognition form" }),
      /* @__PURE__ */ jsx("p", { className: "muted", children: "Validation-only request using Precognition headers." }),
      /* @__PURE__ */ jsxs("form", { className: "form-grid", onSubmit: validateForm, children: [
        /* @__PURE__ */ jsxs("label", { className: "field", children: [
          "Name",
          /* @__PURE__ */ jsx(
            "input",
            {
              name: "name",
              value: form.name,
              onChange: handleChange,
              type: "text",
              className: "input"
            }
          ),
          errors.name ? /* @__PURE__ */ jsx("span", { className: "error", children: errors.name[0] }) : null
        ] }),
        /* @__PURE__ */ jsxs("label", { className: "field", children: [
          "Email",
          /* @__PURE__ */ jsx(
            "input",
            {
              name: "email",
              value: form.email,
              onChange: handleChange,
              type: "email",
              className: "input"
            }
          ),
          errors.email ? /* @__PURE__ */ jsx("span", { className: "error", children: errors.email[0] }) : null
        ] }),
        /* @__PURE__ */ jsxs("div", { className: "actions", children: [
          /* @__PURE__ */ jsx("button", { type: "submit", className: "btn", children: "Validate only" }),
          status ? /* @__PURE__ */ jsx("span", { className: "status", children: status }) : null
        ] })
      ] })
    ] })
  ] });
}
const __vite_glob_0_2 = /* @__PURE__ */ Object.freeze(/* @__PURE__ */ Object.defineProperty({
  __proto__: null,
  default: Settings
}, Symbol.toStringTag, { value: "Module" }));
const sortOptions = [
  { value: "name", label: "Name ↑" },
  { value: "name_desc", label: "Name ↓" },
  { value: "id_desc", label: "ID ↓" },
  { value: "role", label: "Role" }
];
function Users({ title, users = [], sort, page, totalPages, prevPage, nextPage }) {
  return /* @__PURE__ */ jsxs("div", { children: [
    /* @__PURE__ */ jsx("h1", { children: title }),
    /* @__PURE__ */ jsxs("div", { className: "controls", children: [
      /* @__PURE__ */ jsx("span", { className: "label", children: "Sort:" }),
      sortOptions.map((option) => /* @__PURE__ */ jsx(
        Link,
        {
          href: `/users?sort=${option.value}`,
          className: `chip${sort === option.value ? " active" : ""}`,
          children: option.label
        },
        option.value
      ))
    ] }),
    /* @__PURE__ */ jsx("ul", { className: "user-list", children: users.map((user) => /* @__PURE__ */ jsxs("li", { className: "user-item", children: [
      /* @__PURE__ */ jsx("div", { className: "user-name", children: user.name }),
      /* @__PURE__ */ jsxs("div", { className: "user-meta", children: [
        "#",
        user.id,
        " · ",
        user.role
      ] })
    ] }, user.id)) }),
    /* @__PURE__ */ jsxs("div", { className: "pager", children: [
      prevPage ? /* @__PURE__ */ jsx(Link, { href: `/users?sort=${sort}&page=${prevPage}`, className: "btn secondary", children: "Prev" }) : null,
      nextPage ? /* @__PURE__ */ jsx(
        Link,
        {
          href: `/users?sort=${sort}&page=${nextPage}`,
          className: "btn",
          preserveScroll: true,
          only: ["users", "page", "prevPage", "nextPage", "totalPages"],
          children: "Load more"
        }
      ) : null
    ] }),
    /* @__PURE__ */ jsxs("p", { className: "muted", children: [
      "Page ",
      page,
      " of ",
      totalPages
    ] })
  ] });
}
const __vite_glob_0_3 = /* @__PURE__ */ Object.freeze(/* @__PURE__ */ Object.defineProperty({
  __proto__: null,
  default: Users
}, Symbol.toStringTag, { value: "Module" }));
function Layout({ children }) {
  const page = usePage();
  const menu = page.props.menu ?? [];
  const flash = page.props.flash ?? {};
  return /* @__PURE__ */ jsxs("div", { className: "app-container", children: [
    /* @__PURE__ */ jsx("nav", { children: menu.map((item) => /* @__PURE__ */ jsx(
      Link,
      {
        href: item.href,
        className: page.url === item.href ? "active" : void 0,
        children: item.label
      },
      item.href
    )) }),
    flash.success && /* @__PURE__ */ jsx("div", { className: "flash flash-success", children: flash.success }),
    flash.error && /* @__PURE__ */ jsx("div", { className: "flash flash-error", children: flash.error }),
    /* @__PURE__ */ jsx("main", { className: "card", children })
  ] });
}
const pages = /* @__PURE__ */ Object.assign({ "./Pages/Error.jsx": __vite_glob_0_0, "./Pages/Home.jsx": __vite_glob_0_1, "./Pages/Settings.jsx": __vite_glob_0_2, "./Pages/Users.jsx": __vite_glob_0_3 });
function resolvePage(name) {
  const page = pages[`./Pages/${name}.jsx`];
  if (!page) {
    throw new Error(`Page ${name} not found!`);
  }
  const component = page.default;
  if (component.layout === void 0) {
    component.layout = (pageNode) => /* @__PURE__ */ jsx(Layout, { children: pageNode });
  }
  return component;
}
createServer(
  (page) => createInertiaApp({
    page,
    render: renderToString,
    resolve: resolvePage,
    setup: ({ App, props }) => /* @__PURE__ */ jsx(App, { ...props })
  })
);
