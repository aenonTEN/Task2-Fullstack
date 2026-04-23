import { Injectable } from "@angular/core";
import { HttpClient, HttpHeaders } from "@angular/common/http";
import { firstValueFrom } from "rxjs";

@Injectable({ providedIn: "root" })
export class ApiService {
  token = localStorage.getItem("accessToken") ?? "";
  me: any = null;
  baseUrl = "/api/v1";

  constructor(private http: HttpClient) {}

  getAuthHeaders(): HttpHeaders {
    return new HttpHeaders({
      Authorization: `Bearer ${this.token}`,
      "Content-Type": "application/json"
    });
  }

  async login(username: string, password: string): Promise<any> {
    return firstValueFrom(this.http.post<any>("/api/v1/auth/login", { username, password }));
  }

  async logout(): Promise<any> {
    return firstValueFrom(this.http.post("/api/v1/auth/logout", {}, { headers: this.getAuthHeaders() }));
  }

  async getMe(): Promise<any> {
    return firstValueFrom(this.http.get<any>("/api/v1/me", { headers: this.getAuthHeaders() }));
  }

  async candidates(): Promise<any[]> {
    return firstValueFrom(this.http.get<any[]>("/api/v1/recruitment/candidates", { headers: this.getAuthHeaders() }));
  }

  async createCandidate(data: any): Promise<any> {
    return firstValueFrom(this.http.post<any>("/api/v1/recruitment/candidates", data, { headers: this.getAuthHeaders() }));
  }

  async searchCandidates(params: Record<string, string>): Promise<any[]> {
    const query = new URLSearchParams(params).toString();
    return firstValueFrom(this.http.get<any[]>(`/api/v1/recruitment/search?${query}`, { headers: this.getAuthHeaders() }));
  }

  async bulkImport(candidates: any[]): Promise<any> {
    return firstValueFrom(this.http.post<any>("/api/v1/recruitment/bulk", { candidates }, { headers: this.getAuthHeaders() }));
  }

  async qualifications(): Promise<any[]> {
    return firstValueFrom(this.http.get<any[]>("/api/v1/compliance/qualifications", { headers: this.getAuthHeaders() }));
  }

  async createQualification(data: any): Promise<any> {
    return firstValueFrom(this.http.post<any>("/api/v1/compliance/qualifications", data, { headers: this.getAuthHeaders() }));
  }

  async checkRestriction(candidateId: string): Promise<any> {
    return firstValueFrom(this.http.post<any>("/api/v1/compliance/restrictions/check", { candidateId }, { headers: this.getAuthHeaders() }));
  }

  async applyRestriction(data: any): Promise<any> {
    return firstValueFrom(this.http.post<any>("/api/v1/compliance/restrictions", data, { headers: this.getAuthHeaders() }));
  }

  async expireQualifications(): Promise<any> {
    return firstValueFrom(this.http.post<any>("/api/v1/compliance/qualifications/expire", {}, { headers: this.getAuthHeaders() }));
  }

  async reactivateQualification(data: any): Promise<any> {
    return firstValueFrom(this.http.post<any>("/api/v1/compliance/qualifications/reactivate", data, { headers: this.getAuthHeaders() }));
  }

  async cases(): Promise<any[]> {
    return firstValueFrom(this.http.get<any[]>("/api/v1/cases", { headers: this.getAuthHeaders() }));
  }

  async createCase(data: any): Promise<any> {
    return firstValueFrom(this.http.post<any>("/api/v1/cases", data, { headers: this.getAuthHeaders() }));
  }

  async caseAttachments(caseId: string): Promise<any[]> {
    return firstValueFrom(this.http.get<any[]>(`/api/v1/cases/${caseId}/attachments`, { headers: this.getAuthHeaders() }));
  }

  async uploadAttachment(caseId: string, file: File): Promise<any> {
    const initRes = await firstValueFrom(this.http.post<any>("/api/v1/attachments/init", {
      caseId,
      fileName: file.name,
      fileSize: file.size,
      mimeType: file.type || "application/octet-stream"
    }, { headers: this.getAuthHeaders() }));

    const chunkSize = 1024 * 1024;
    const totalChunks = Math.ceil(file.size / chunkSize);
    const uploadId = initRes.uploadId;

    for (let i = 0; i < totalChunks; i++) {
      const start = i * chunkSize;
      const end = Math.min(start + chunkSize, file.size);
      const chunk = file.slice(start, end);
      const buffer = await chunk.arrayBuffer();
      const chunkArray = new Uint8Array(buffer);
      const hexChunk = Array.from(chunkArray).map(b => b.toString(16).padStart(2, "0")).join("");
      await firstValueFrom(this.http.post<any>(`/api/v1/attachments/${uploadId}/chunk`, {
        uploadId,
        chunkData: hexChunk,
        chunkIndex: i
      }, { headers: this.getAuthHeaders() }));
    }

    return firstValueFrom(this.http.post<any>("/api/v1/attachments/complete", {
      uploadId,
      caseId
    }, { headers: this.getAuthHeaders() }));
  }

  async attachmentDownloadUrl(attachmentId: string): Promise<string> {
    return `${this.baseUrl}/attachments/${attachmentId}/download?token=${this.token}`;
  }

  async updateCaseStatus(caseId: string, status: string): Promise<any> {
    return firstValueFrom(this.http.patch<any>(`/api/v1/cases/${caseId}/status`, { status }, { headers: this.getAuthHeaders() }));
  }

  async assignCase(caseId: string, assignedTo: string, note?: string): Promise<any> {
    return firstValueFrom(this.http.post<any>(`/api/v1/cases/${caseId}/assign`, { assignedTo, note }, { headers: this.getAuthHeaders() }));
  }

  async caseHistory(caseId: string): Promise<any[]> {
    return firstValueFrom(this.http.get<any[]>(`/api/v1/cases/${caseId}/history`, { headers: this.getAuthHeaders() }));
  }

  async positions(): Promise<any[]> {
    return firstValueFrom(this.http.get<any[]>("/api/v1/positions", { headers: this.getAuthHeaders() }));
  }

  async createPosition(data: any): Promise<any> {
    return firstValueFrom(this.http.post<any>("/api/v1/positions", data, { headers: this.getAuthHeaders() }));
  }

  async closePosition(positionId: string): Promise<any> {
    return firstValueFrom(this.http.post<any>(`/api/v1/positions/${positionId}/close`, {}, { headers: this.getAuthHeaders() }));
  }

  async qualificationProfiles(): Promise<any[]> {
    return firstValueFrom(this.http.get<any[]>("/api/v1/profiles/qualifications", { headers: this.getAuthHeaders() }));
  }

  async createQualificationProfile(data: any): Promise<any> {
    return firstValueFrom(this.http.post<any>("/api/v1/profiles/qualifications", data, { headers: this.getAuthHeaders() }));
  }

  async auditRecords(): Promise<any[]> {
    return firstValueFrom(this.http.get<any[]>("/api/v1/audit/records", { headers: this.getAuthHeaders() }));
  }

  async tags(): Promise<any[]> {
    return firstValueFrom(this.http.get<any[]>("/api/v1/tags", { headers: this.getAuthHeaders() }));
  }

  async createTag(data: { name: string; color?: string; description?: string }): Promise<any> {
    return firstValueFrom(this.http.post<any>("/api/v1/tags", data, { headers: this.getAuthHeaders() }));
  }

  async deleteTag(tagId: string): Promise<any> {
    return firstValueFrom(this.http.delete<any>(`/api/v1/tags/${tagId}`, { headers: this.getAuthHeaders() }));
  }

  async assignTags(entityType: string, entityId: string, tagIds: string[]): Promise<any> {
    return firstValueFrom(this.http.post<any>("/api/v1/tags/assign", {
      entityType,
      entityId,
      tagIds
    }, { headers: this.getAuthHeaders() }));
  }

  async getTagsForEntity(entityType: string, entityId: string): Promise<any[]> {
    return firstValueFrom(this.http.get<any[]>(`/api/v1/tags/entity?entityType=${entityType}&entityId=${entityId}`, { headers: this.getAuthHeaders() }));
  }
}