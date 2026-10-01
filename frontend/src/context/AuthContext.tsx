"use client";

import React, { createContext, useContext, useState, useEffect } from "react";
import { UserProfile } from "@/lib/api";

interface AuthContextType {
  user: UserProfile | null;
  loading: boolean;
  loginAs: (role: "student_active" | "student_guest" | "admin") => void;
  logout: () => void;
}

const DEMO_USERS: Record<string, UserProfile> = {
  student_active: {
    id: 101,
    telegram_id: 549102941,
    username: "student_spark",
    first_name: "Алексей",
    last_name: "Морозов",
    photo_url: "",
    role: "student",
    has_access: true,
    access_granted_at: "2026-09-20",
  },
  student_guest: {
    id: 102,
    telegram_id: 391029410,
    username: "guest_candidate",
    first_name: "Екатерина",
    last_name: "Павлова",
    photo_url: "",
    role: "student",
    has_access: false,
  },
  admin: {
    id: 1,
    telegram_id: 891230491,
    username: "spark_admin",
    first_name: "так называемый Иль",
    last_name: "",
    photo_url: "",
    role: "admin",
    has_access: true,
    access_granted_at: "2026-09-01",
  },
};

const AuthContext = createContext<AuthContextType>({
  user: null,
  loading: true,
  loginAs: () => {},
  logout: () => {},
});

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<UserProfile | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    // Check local storage for session
    const saved = localStorage.getItem("spark_active_user");
    if (saved) {
      try {
        setUser(JSON.parse(saved));
      } catch {
        setUser(DEMO_USERS.student_active);
      }
    } else {
      // Default to active student
      setUser(DEMO_USERS.student_active);
      localStorage.setItem("spark_active_user", JSON.stringify(DEMO_USERS.student_active));
    }
    setLoading(false);
  }, []);

  const loginAs = (role: "student_active" | "student_guest" | "admin") => {
    const selected = DEMO_USERS[role];
    setUser(selected);
    localStorage.setItem("spark_active_user", JSON.stringify(selected));
  };

  const logout = () => {
    setUser(null);
    localStorage.removeItem("spark_active_user");
  };

  return (
    <AuthContext.Provider value={{ user, loading, loginAs, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  return useContext(AuthContext);
}
