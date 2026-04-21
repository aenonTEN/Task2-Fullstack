import { Component } from "@angular/core";
import { CommonModule } from "@angular/common";
import { FormsModule } from "@angular/forms";
import { ApiService } from "../api.service";

@Component({
  selector: "app-compliance",
  standalone: true,
  imports: [CommonModule, FormsModule],
  template: `
    <div class="section">
      <h2>Compliance</h2>
      
      <h3>Create Qualification</h3>
      <div class="grid">
        <label><span>Candidate ID</span><input [(ngModel)]="qualCandidateId" /></label>
        <label><span>Name</span><input [(ngModel)]="qualName" /></label>
        <label><span>Issued Date (YYYY-MM-DD)</span><input [(ngModel)]="qualIssuedDate" /></label>
        <label><span>Expiry Date (YYYY-MM-DD)</span><input [(ngModel)]="qualExpiryDate" /></label>
      </div>
      <div class="row">
        <button (click)="createQualification()">Create</button>
        <button (click)="loadQualifications()">Refresh</button>
      </div>
      <p *ngIf="qualMessage" class="message">{{ qualMessage }}</p>

      <h3>Qualifications</h3>
      <pre class="message">{{ qualifications | json }}</pre>

      <h3>Check Restriction</h3>
      <div class="grid">
        <label><span>Candidate ID</span><input [(ngModel)]="checkCandidateId" /></label>
      </div>
      <div class="row">
        <button (click)="checkRestriction()">Check</button>
      </div>
      <pre *ngIf="restrictionResult" class="message">{{ restrictionResult | json }}</pre>

      <h3>Apply Restriction</h3>
      <div class="grid">
        <label><span>Candidate ID</span><input [(ngModel)]="restrictCandidateId" /></label>
        <label><span>Type</span><input [(ngModel)]="restrictType" placeholder="purchase_168h" /></label>
        <label><span>Reason</span><input [(ngModel)]="restrictReason" /></label>
        <label><span>Window Hours</span><input [(ngModel)]="restrictHours" type="number" placeholder="168" /></label>
      </div>
      <div class="row">
        <button (click)="applyRestriction()">Apply</button>
      </div>
      <p *ngIf="restrictMessage" class="message">{{ restrictMessage }}</p>
    </div>
  `
})
export class ComplianceComponent {
  qualifications: any[] = [];
  qualCandidateId = "";
  qualName = "";
  qualIssuedDate = "";
  qualExpiryDate = "";
  qualMessage = "";
  checkCandidateId = "";
  restrictionResult: any = null;
  restrictCandidateId = "";
  restrictType = "purchase_168h";
  restrictReason = "";
  restrictHours: number | null = 168;
  restrictMessage = "";

  constructor(private api: ApiService) {}

  async loadQualifications() {
    this.qualifications = await this.api.qualifications();
  }

  async createQualification() {
    const result = await this.api.createQualification({
      candidateId: this.qualCandidateId,
      name: this.qualName,
      issuedDate: this.qualIssuedDate,
      expiryDate: this.qualExpiryDate || undefined
    });
    this.qualMessage = "Created: " + result.id;
    this.qualCandidateId = "";
    this.qualName = "";
    this.qualIssuedDate = "";
    this.qualExpiryDate = "";
    await this.loadQualifications();
  }

  async checkRestriction() {
    this.restrictionResult = await this.api.checkRestriction(this.checkCandidateId);
  }

  async applyRestriction() {
    const result = await this.api.applyRestriction({
      candidateId: this.restrictCandidateId,
      restrictionType: this.restrictType,
      reason: this.restrictReason,
      windowHours: this.restrictHours ?? 168
    });
    this.restrictMessage = "Applied: " + result.id;
    this.restrictCandidateId = "";
    this.restrictType = "purchase_168h";
    this.restrictReason = "";
  }
}