import React from "react";
import { createRoot } from "react-dom/client";
import App from "./App";
import { LanguageProvider } from "./i18n";
import { ShopBagProvider } from "./ShopBag";
import "./styles.css";
createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <LanguageProvider>
      <ShopBagProvider>
        <App />
      </ShopBagProvider>
    </LanguageProvider>
  </React.StrictMode>,
);
