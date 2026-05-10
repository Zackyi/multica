"use client";

import { useEffect, type ReactNode } from "react";
import { getApi } from "../api";
import { useAuthStore } from "../auth";
import { useWorkspaceStore } from "../workspace";
import { createLogger } from "../logger";
import { defaultStorage } from "./storage";
import type { StorageAdapter } from "../types/storage";
import type { Workspace } from "../types";

const logger = createLogger("auth");

// Local mode default user
const LOCAL_MODE_USER = {
  id: "local-user",
  name: "Local User",
  email: "local@multica.local",
  avatar_url: null as string | null,
  created_at: new Date().toISOString(),
  updated_at: new Date().toISOString(),
};

type MaybeUser = { user: typeof LOCAL_MODE_USER | null; wsList: any[] | null };

export function AuthInitializer({
  children,
  onLogin,
  onLogout,
  storage = defaultStorage,
}: {
  children: ReactNode;
  onLogin?: () => void;
  onLogout?: () => void;
  storage?: StorageAdapter;
}) {
  useEffect(() => {
    const api = getApi();
    const wsId = storage.getItem("multica_workspace_id");

    // Check if we're in local mode by calling /health
    // Use relative path to go through Next.js rewrite (avoid CORS on cross-origin requests)
    fetch("/health")
      .then((res) => res.json() as Promise<{ local_mode?: string; status?: string }>)
      .then((data) => {
        if (data.local_mode === "enabled") {
          logger.info("local mode detected, skipping authentication");
          // Set local mode user without auth
          useAuthStore.setState({
            user: LOCAL_MODE_USER,
            isLoading: false,
          });
          onLogin?.();
          // Load workspaces using relative path (no auth) in local mode
          return fetch("/api/workspaces")
            .then((res) => res.json() as Promise<Workspace[]>)
            .then((wsList) => {
              if (wsList && wsList.length > 0) {
                useWorkspaceStore.getState().hydrateWorkspace(wsList, wsId);
              }
              return { user: LOCAL_MODE_USER, wsList };
            });
        }
        // Not local mode
        return Promise.reject(new Error("not local mode"));
      })
      .catch(() => {
        // Normal auth flow
        const token = storage.getItem("multica_token");
        if (!token) {
          onLogout?.();
          useAuthStore.setState({ isLoading: false });
          return Promise.reject({ user: null, wsList: null });
        }

        api.setToken(token);

        return Promise.all([api.getMe(), api.listWorkspaces()])
          .then(([user, wsList]) => ({ user, wsList }));
      })
      .then((result: MaybeUser) => {
        const { user, wsList } = result;
        if (user) {
          useAuthStore.setState({ user, isLoading: false });
        }
        if (wsList && wsList.length > 0) {
          useWorkspaceStore.getState().hydrateWorkspace(wsList, wsId);
        }
      })
      .catch((err) => {
        logger.error("auth init failed", err);
        api.setToken(null);
        api.setWorkspaceId(null);
        storage.removeItem("multica_token");
        storage.removeItem("multica_workspace_id");
        onLogout?.();
        useAuthStore.setState({ user: null, isLoading: false });
      });
  }, []);

  return <>{children}</>;
}
