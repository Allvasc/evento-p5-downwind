import { defineStore } from "pinia";
import { ref } from "vue";
import { TEAM_TOKEN_KEY } from "@/lib/teamApi";

export const useTeamAuthStore = defineStore("team-auth", () => {
  const token = ref<string | null>(localStorage.getItem(TEAM_TOKEN_KEY));
  const role = ref<string | null>(localStorage.getItem("p5_team_role"));
  const name = ref<string | null>(localStorage.getItem("p5_team_name"));
  const vendorId = ref<string | null>(localStorage.getItem("p5_team_vendor_id"));
  const vendorName = ref<string | null>(localStorage.getItem("p5_team_vendor_name"));

  function setSession(newToken: string, newRole: string, newName: string, newVendorId = "", newVendorName = "") {
    token.value = newToken;
    role.value = newRole;
    name.value = newName;
    vendorId.value = newVendorId || null;
    vendorName.value = newVendorName || null;
    localStorage.setItem(TEAM_TOKEN_KEY, newToken);
    localStorage.setItem("p5_team_role", newRole);
    localStorage.setItem("p5_team_name", newName);
    if (newVendorId) localStorage.setItem("p5_team_vendor_id", newVendorId);
    else localStorage.removeItem("p5_team_vendor_id");
    if (newVendorName) localStorage.setItem("p5_team_vendor_name", newVendorName);
    else localStorage.removeItem("p5_team_vendor_name");
  }

  function logout() {
    token.value = null;
    role.value = null;
    name.value = null;
    vendorId.value = null;
    vendorName.value = null;
    localStorage.removeItem(TEAM_TOKEN_KEY);
    localStorage.removeItem("p5_team_role");
    localStorage.removeItem("p5_team_name");
    localStorage.removeItem("p5_team_vendor_id");
    localStorage.removeItem("p5_team_vendor_name");
  }

  return { token, role, name, vendorId, vendorName, setSession, logout };
});
