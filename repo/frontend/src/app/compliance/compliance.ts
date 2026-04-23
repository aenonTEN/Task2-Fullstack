import { Component, OnInit } from "@angular/core";
import { CommonModule } from "@angular/common";
import { FormsModule } from "@angular/forms";
import { ApiService } from "../api.service";

@Component({
  selector: "app-compliance",
  standalone: true,
  imports: [CommonModule, FormsModule],
  template: `
    <div class="section">
      <div class="section-header">
        <h2 class="section-title">Compliance</h2>
      </div>
      
      <div class="glass-panel">
        <h3>Create Qualification</h3>
        <div class="grid">
          <label><span>Candidate ID</span><input [(ngModel)]="qualCandidateId" /></label>
          <label><span>Name</span><input [(ngModel)]="qualName" /></label>
          <label><span>Issued Date</span><input [(ngModel)]="qualIssuedDate" placeholder="YYYY-MM-DD" /></label>
          <label><span>Expiry Date</span><input [(ngModel)]="qualExpiryDate" placeholder="YYYY-MM-DD" /></label>
        </div>
        <div class="row">
          <button (click)="createQualification()" [disabled]="loading">Create</button>
          <button class="btn-secondary" (click)="loadQualifications()" [disabled]="loading">Refresh</button>
        </div>
      </div>

      <div class="glass-panel">
        <div class="section-header">
          <h3>Qualifications</h3>
          <span class="badge">{{ qualifications.length }}</span>
        </div>
        
        <div *ngIf="loading" class="loading">
          <span class="spinner"></span>Loading...
        </div>
        
        <div *ngIf="!loading && qualifications.length === 0" class="empty-state">
          <div class="empty-state-icon">📜</div>
          <div class="empty-state-title">No qualifications</div>
        </div>
        
        <div *ngIf="!loading && qualifications.length > 0" class="card-grid">
          <div *ngFor="let q of qualifications" class="card">
            <div class="card-header">
              <span class="card-title">{{ q.name }}</span>
              <span class="badge" [class.active]="q.status === 'active'" [class.inactive]="q.status !== 'active'">
                {{ q.status }}
              </span>
            </div>
            <div class="card-body">
              <p><strong>Candidate:</strong> {{ q.candidateId }}</p>
              <p><strong>Issued:</strong> {{ q.issuedDate || '-' }}</p>
              <p><strong>Expires:</strong> {{ q.expiryDate || 'Never' }}</p>
            </div>
          </div>
        </div>
      </div>

      <div class="glass-panel">
        <h3>Check Restriction</h3>
        <div class="grid">
          <label><span>Candidate ID</span><input [(ngModel)]="checkCandidateId" /></label>
        </div>
        <div class="row">
          <button (click)="checkRestriction()" [disabled]="checking">Check</button>
        </div>
        
        <div *ngIf="checking" class="loading">
          <span class="spinner"></span>Checking...
        </div>
        
        <div *ngIf="restrictionResult" class="card" style="margin-top: 1rem;">
          <div class="card-body">
            <p><strong>Allowed:</strong> {{ restrictionResult.allowed }}</p>
            <p *ngIf="restrictionResult.reason"><strong>Reason:</strong> {{ restrictionResult.reason }}</p>
          </div>
        </div>
      </div>

      <div class="glass-panel">
        <h3>Apply Restriction</h3>
        <div class="grid">
          <label><span>Candidate ID</span><input [(ngModel)]="restrictCandidateId" /></label>
          <label><span>Type</span><input [(ngModel)]="restrictType" placeholder="purchase_168h" /></label>
          <label><span>Reason</span><input [(ngModel)]="restrictReason" /></label>
          <label><span>Hours</span><input [(ngModel)]="restrictHours" type="number" placeholder="168" /></label>
        </div>
        <div class="row">
          <button class="btn-danger" (click)="applyRestriction()" [disabled]="loading">Apply</button>
        </div>
      </div>
    </div>
  `
})
export class ComplianceComponent implements OnInit {
  qualifications: any[] = [];
  qualCandidateId = "";
  qualName = "";
  qualIssuedDate = "";
  qualExpiryDate = "";
  checkCandidateId = "";
  restrictionResult: any = null;
  restrictCandidateId = "";
  restrictType = "purchase_168h";
  restrictReason = "";
  restrictHours = 168;
  loading = false;
  checking = false;

  constructor(private api: ApiService) {}

  ngOnInit() {
    this.loadQualifications();
  }

  async loadQualifications() {
    this.loading = true;
    try {
      this.qualifications = await this.api.qualifications();
    } catch (e) {
      console.error(e);
    }
    this.loading = false;
  }

  async createQualification() {
    this.loading = true;
    try {
      await this.api.createQualification({
        candidateId: this.qualCandidateId,
        name: this.qualName,
        issuedDate: this.qualIssuedDate,
        expiryDate: this.qualExpiryDate || undefined
      });
      this.qualCandidateId = "";
      this.qualName = "";
      this.qualIssuedDate = "";
      this.qualExpiryDate = "";
      await this.loadQualifications();
    } catch (e) {
      console.error(e);
    }
    this.loading = false;
  }

  async checkRestriction() {
    this.checking = true;
    this.restrictionResult = null;
    try {
      this.restrictionResult = await this.api.checkRestriction(this.checkCandidateId);
    } catch (e) {
      console.error(e);
    }
    this.checking = false;
  }

  async applyRestriction() {
    this.loading = true;
    try {
      await this.api.applyRestriction({
        candidateId: this.restrictCandidateId,
        restrictionType: this.restrictType,
        reason: this.restrictReason,
        windowHours: this.restrictHours
      });
      this.restrictCandidateId = "";
      this.restrictReason = "";
    } catch (e) {
      console.error(e);
    }
    this.loading = false;
  }
}