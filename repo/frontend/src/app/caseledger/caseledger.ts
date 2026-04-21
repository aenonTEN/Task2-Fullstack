import { Component } from "@angular/core";
import { CommonModule } from "@angular/common";
import { FormsModule } from "@angular/forms";
import { ApiService } from "../api.service";

@Component({
  selector: "app-caseledger",
  standalone: true,
  imports: [CommonModule, FormsModule],
  template: `
    <div class="section">
      <h2>Case Ledger</h2>
      
      <h3>Create Case</h3>
      <div class="grid">
        <label><span>Candidate ID</span><input [(ngModel)]="caseCandidateId" /></label>
        <label><span>Case Type</span><input [(ngModel)]="caseType" placeholder="enrollment" /></label>
        <label><span>Subject</span><input [(ngModel)]="caseSubject" /></label>
        <label style="grid-column: 1 / -1;"><span>Description</span><input [(ngModel)]="caseDescription" /></label>
      </div>
      <div class="row">
        <button (click)="createCase()">Create</button>
        <button (click)="loadCases()">Refresh</button>
      </div>
      <p *ngIf="caseMessage" class="message">{{ caseMessage }}</p>

      <h3>Cases</h3>
      <div class="cases-grid">
        <div *ngFor="let c of cases" class="case-card">
          <div class="case-header">
            <strong>{{ c.caseNumber }}</strong>
            <span class="badge">{{ c.status }}</span>
          </div>
          <div class="case-body">
            <p><strong>Type:</strong> {{ c.caseType }}</p>
            <p><strong>Subject:</strong> {{ c.subject }}</p>
            <p><strong>Description:</strong> {{ c.description }}</p>
            <p><strong>Candidate:</strong> {{ c.candidateId }}</p>
          </div>
          <div class="case-attachments">
            <h4>Attachments</h4>
            <div *ngIf="caseAttachments[c.id]?.length === 0">No attachments</div>
            <ul *ngIf="caseAttachments[c.id]?.length">
              <li *ngFor="let att of caseAttachments[c.id]">
                <span>{{ att.fileName }}</span>
                <button (click)="downloadAttachment(att.id, att.fileName)">Download</button>
              </li>
            </ul>
            <div class="upload-row">
              <input type="file" (change)="onAttachmentSelected($event, c.id)" />
              <button (click)="uploadAttachment(c.id)" [disabled]="!pendingAttachment || uploadingAttachment === c.id">
                {{ uploadingAttachment === c.id ? 'Uploading...' : 'Upload' }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  `
})
export class CaseledgerComponent {
  cases: any[] = [];
  caseCandidateId = "";
  caseType = "";
  caseSubject = "";
  caseDescription = "";
  caseMessage = "";
  caseAttachments: Record<string, any[]> = {};
  pendingAttachment: File | null = null;
  pendingCaseId: string | null = null;
  uploadingAttachment: string | null = null;

  constructor(private api: ApiService) {}

  async loadCases() {
    this.cases = await this.api.cases();
    for (const c of this.cases) {
      this.loadAttachments(c.id);
    }
  }

  async loadAttachments(caseId: string) {
    try {
      const atts = await this.api.caseAttachments(caseId);
      this.caseAttachments[caseId] = atts || [];
    } catch {
      this.caseAttachments[caseId] = [];
    }
  }

  onAttachmentSelected(event: Event, caseId: string) {
    const input = event.target as HTMLInputElement;
    this.pendingAttachment = input.files?.[0] || null;
    this.pendingCaseId = caseId;
  }

  async uploadAttachment(caseId: string) {
    if (!this.pendingAttachment || this.uploadingAttachment) return;
    this.uploadingAttachment = caseId;
    try {
      await this.api.uploadAttachment(caseId, this.pendingAttachment);
      this.pendingAttachment = null;
      await this.loadAttachments(caseId);
    } catch (e: any) {
      alert("Upload failed: " + (e.message || "Unknown error"));
    }
    this.uploadingAttachment = null;
  }

  async downloadAttachment(attachmentId: string, fileName: string) {
    const url = await this.api.attachmentDownloadUrl(attachmentId);
    if (url) {
      const a = document.createElement("a");
      a.href = url;
      a.download = fileName;
      a.click();
    }
  }

  async createCase() {
    const result = await this.api.createCase({
      candidateId: this.caseCandidateId,
      caseType: this.caseType,
      subject: this.caseSubject,
      description: this.caseDescription
    });
    this.caseMessage = "Created: " + result.caseNumber;
    this.caseCandidateId = "";
    this.caseType = "";
    this.caseSubject = "";
    this.caseDescription = "";
    await this.loadCases();
  }
}