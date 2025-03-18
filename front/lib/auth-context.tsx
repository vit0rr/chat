"use client";

import {
  createContext,
  useContext,
  useState,
  useEffect,
  ReactNode,
} from "react";
import { useRouter, usePathname } from "next/navigation";
import { Loader2 } from "lucide-react";

export type User = {
  id: string;
  nickname: string;
};

type AuthContextType = {
  user: User | null;
  token: string | null;
  login: (token: string, userId: string, nickname: string) => void;
  logout: () => void;
  isAuthenticated: boolean;
};

const AuthContext = createContext<AuthContextType>({
  user: null,
  token: null,
  login: () => {},
  logout: () => {},
  isAuthenticated: false,
});

export const useAuth = () => useContext(AuthContext);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [token, setToken] = useState<string | null>(null);
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const router = useRouter();
  const pathname = usePathname();

  useEffect(() => {
    const checkAuth = () => {
      const storedToken = localStorage.getItem("token");
      const storedUser = localStorage.getItem("user");

      if (storedToken && storedUser) {
        try {
          const userData = JSON.parse(storedUser);
          if (userData && userData.id && userData.nickname) {
            setUser(userData);
            setToken(storedToken);
            setIsAuthenticated(true);
          } else {
            throw new Error("Invalid user data");
          }
        } catch (error) {
          console.error("Error parsing stored user data:", error);
          localStorage.removeItem("token");
          localStorage.removeItem("user");
          setIsAuthenticated(false);
          setUser(null);
          setToken(null);
        }
      } else {
        setIsAuthenticated(false);
        setUser(null);
        setToken(null);
      }
      setIsLoading(false);
    };

    checkAuth();
  }, []);

  // Handle protected routes
  useEffect(() => {
    if (isLoading) return;

    const publicRoutes = ["/", "/login", "/register"];
    const isPublicRoute = publicRoutes.includes(pathname);

    if (!isAuthenticated && !isPublicRoute) {
      router.push("/login");
    } else if (
      isAuthenticated &&
      (pathname === "/login" || pathname === "/register")
    ) {
      router.push("/rooms");
    }
  }, [isAuthenticated, pathname, router, isLoading, token]);

  const login = (token: string, userId: string, nickname: string) => {
    const userData = {
      id: userId,
      nickname: nickname,
    };

    // Update state
    setUser(userData);
    setToken(token);
    setIsAuthenticated(true);

    // Store in localStorage
    localStorage.setItem("token", token);
    localStorage.setItem("user", JSON.stringify(userData));
  };

  const logout = () => {
    // Clear state
    setUser(null);
    setToken(null);
    setIsAuthenticated(false);

    // Clear localStorage
    localStorage.removeItem("token");
    localStorage.removeItem("user");

    // Redirect to login
    router.push("/login");
  };

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <Loader2 className="w-10 h-10 animate-spin" />
      </div>
    );
  }

  return (
    <AuthContext.Provider
      value={{ user, token, login, logout, isAuthenticated }}
    >
      {children}
    </AuthContext.Provider>
  );
}
