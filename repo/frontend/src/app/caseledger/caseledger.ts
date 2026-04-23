import { Component, OnInit } from "@angular/core";
import { CommonModule } from "@angular/common";
import { FormsModule } from "@angular/forms";
import { ApiService } from "../api.service";

@Component({
  selector: "app-caseledger",
  standalone: true,
  imports: [CommonModule, FormsModule],
  template: `
    <div class="section">
      <div class="section-header">
        <h2 class="section-title">Case Ledger</h2>
      </div>
      
      <div class="glass-panel">
        <h3>Create Case</h3>
        <div class="grid">
          <label><span>Candidate ID</span><input [(ngModel)]="caseCandidateId" /></label>
          <label><span>Case Type</span><input [(ngModel)]="caseType" placeholder="enrollment" /></label>
          <label><span>Subject</span><input [(ngModel)]="caseSubject" /></label>
          <label style="grid-column: 1 / -1;"><span>Description</span><input [(ngModel)]="caseDescription" /></label>
        </div>
        <div class="row">
          <button (click)="createCase()" [disabled]="loading">Create</button>
          <button class="btn-secondary" (click)="loadCases()" [disabled]="loading">Refresh</button>
        </div>
      </div>

      <div class="glass-panel">
        <div class="section-header">
          <h3>Cases</h3>
          <span class="badge">{{ cases.length }}</span>
        </div>
        
        <div *ngIf="loading" class="loading">
          <span class="spinner"></span>Loading...
        </div>
        
        <div *ngIf="!loading && cases.length === 0" class="empty-state">
          <div class="empty-state-icon">📁</div>
          <div class="empty-state-title">No cases</div>
        </div>
        
        <div *ngIf="!loading && cases.length > 0" class="card-grid">
          <div *ngFor="let c of cases" class="card">
            <div class="card-header">
              <span class="card-title">{{ c.caseNumber }}</span>
              <select [(ngModel)]="c.status" (change)="updateStatus(c.id, c.status)">
                <option value="open">Open</option>
                <option value="in_progress">In Progress</option>
                <option value="pending">Pending</option>
                <option value="resolved">Resolved</option>
                <option value="closed">Closed</option>
              </select>
            </div>
            <div class="card-body">
              <p><strong>Type:</strong> {{ c.caseType }}</p>
              <p><strong>Subject:</strong> {{ c.subject }}</p>
              <p><strong>Candidate:</strong> {{ c.candidateId }}</p>
              <p *ngIf="c.description"><strong>Description:</strong> {{ c.description }}</p>
            </div>
            <div class="card-actions">
              <input [(ngModel)]="c.assignedTo" placeholder="Assign to..." style="flex:1" />
              <button class="btn-small" (click)="assignCase(c.id, c.assignedTo)">Assign</button>
            </div>
          </div>
        </div>
      </div>
    </div>
  `
})
export class CaseledgerComponent implements OnInit {
  cases: any[] = [];
  caseCandidateId = "";
  caseType = "";
  caseSubject = "";
  caseDescription = "";
  loading = false;

  constructor(private api: ApiService) {}

  ngOnInit() {
    this.loadCases();
  }

  async loadCases() {
    this.loading = true;
    try {
      this.cases = await this.api.cases();
    } catch (e) {
      console.error(e);
    }
    this.loading = false;
  }

  async createCase() {
    this.loading = true;
    try {
      await this.api.createCase({
        candidateId: this.caseCandidateId,
        caseType: this.caseType,
        subject: this.caseSubject,
        description: this.caseDescription
      });
      this.caseCandidateId = "";
      this.caseType = "";
      this.caseSubject = "";
      this.caseDescription = "";
      await this.loadCases();
    } catch (e) {
      console.error(e);
    }
    this.loading = false;
  }

  async updateStatus(caseId: string, status: string) {
    try {
      await this.api.updateCaseStatus(caseId, status);
    } catch (e) {
      console.error(e);
    }
  }

  async assignCase(caseId: string, assignedTo: string) {
    if (!assignedTo) return;
    try {
      await this.api.assignCase(caseId, assignedTo);
    } catch (e) {
      console.error(e);
    }
  }
}