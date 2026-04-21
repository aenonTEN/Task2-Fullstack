import { Component } from "@angular/core";
import { CommonModule } from "@angular/common";
import { FormsModule } from "@angular/forms";
import { ApiService } from "../api.service";

@Component({
  selector: "app-recruitment",
  standalone: true,
  imports: [CommonModule, FormsModule],
  template: `
    <div class="section">
      <h2>Recruitment</h2>
      
      <h3>Create Candidate</h3>
      <div class="grid">
        <label><span>Name</span><input [(ngModel)]="candName" /></label>
        <label><span>Phone</span><input [(ngModel)]="candPhone" /></label>
        <label><span>ID Number</span><input [(ngModel)]="candIdNumber" /></label>
        <label><span>Education</span><input [(ngModel)]="candEducation" /></label>
        <label><span>Experience Years</span><input [(ngModel)]="candExperienceYears" type="number" /></label>
        <label style="grid-column: 1 / -1;"><span>Skills (comma-separated)</span><input [(ngModel)]="candSkills" /></label>
      </div>
      <div class="row">
        <button (click)="createCandidate()">Create</button>
        <button (click)="loadCandidates()">Refresh</button>
      </div>
      <p *ngIf="candidateResult" class="message">{{ candidateResult }}</p>

      <h3>Bulk Import (JSON)</h3>
      <p class="hint">Upload a JSON file with array of candidates: [{"name":"John","phone":"123","idNumber":"ID1","education":"BS","experienceYears":5,"skills":["java"]}]</p>
      <div class="row">
        <input type="file" (change)="onBulkFileSelected($event)" accept=".json" #bulkFileInput />
        <button (click)="bulkImport()" [disabled]="!bulkFile || bulkImporting">{{ bulkImporting ? 'Importing...' : 'Import' }}</button>
      </div>
      <div *ngIf="bulkProgress" class="progress">{{ bulkProgress }}</div>
      <div *ngIf="bulkResult">
        <p class="message">Imported: {{ bulkResult.importedCount }}, Errors: {{ bulkResult.errors?.length || 0 }}</p>
        <div *ngIf="bulkResult.errors?.length" class="errors">
          <h4>Errors:</h4>
          <ul>
            <li *ngFor="let err of bulkResult.errors">{{ err.index }}: {{ err.error }}</li>
          </ul>
        </div>
      </div>

      <h3>Candidates</h3>
      <pre class="message">{{ candidates | json }}</pre>

      <h3>Search</h3>
      <div class="grid">
        <label><span>Keyword</span><input [(ngModel)]="searchKeyword" /></label>
        <label><span>Skill</span><input [(ngModel)]="searchSkill" /></label>
        <label><span>Education</span><input [(ngModel)]="searchEducation" /></label>
      </div>
      <div class="row">
        <button (click)="searchCandidates()">Search</button>
      </div>
      <pre class="message">{{ searchResults | json }}</pre>
    </div>
  `
})
export class RecruitmentComponent {
  candidates: any[] = [];
  candName = "";
  candPhone = "";
  candIdNumber = "";
  candEducation = "";
  candExperienceYears: number | null = null;
  candSkills = "";
  candidateResult = "";
  searchKeyword = "";
  searchSkill = "";
  searchEducation = "";
  searchResults: any[] = [];
  bulkFile: File | null = null;
  bulkImporting = false;
  bulkProgress = "";
  bulkResult: any = null;

  constructor(private api: ApiService) {}

  onBulkFileSelected(event: Event) {
    const input = event.target as HTMLInputElement;
    this.bulkFile = input.files?.[0] || null;
    this.bulkResult = null;
  }

  async bulkImport() {
    if (!this.bulkFile) return;
    this.bulkImporting = true;
    this.bulkProgress = "Reading file...";
    this.bulkResult = null;

    try {
      const text = await this.bulkFile.text();
      this.bulkProgress = "Parsing...";
      const candidates = JSON.parse(text);
      this.bulkProgress = `Importing ${candidates.length} candidates...`;
      const result = await this.api.bulkImport(candidates);
      this.bulkResult = result;
      this.bulkProgress = "";
      await this.loadCandidates();
    } catch (e: any) {
      this.bulkResult = { errors: [{ index: -1, error: e.message || "Import failed" }] };
      this.bulkProgress = "";
    }
    this.bulkImporting = false;
  }

  async loadCandidates() {
    this.candidates = await this.api.candidates();
  }

  async createCandidate() {
    const skills = this.candSkills.split(",").map((s: string) => s.trim()).filter(Boolean);
    const result = await this.api.createCandidate({
      name: this.candName,
      phone: this.candPhone,
      idNumber: this.candIdNumber,
      education: this.candEducation,
      experienceYears: this.candExperienceYears ?? 0,
      skills
    });
    this.candidateResult = result.isNew ? "Created: " + result.candidateId : "Duplicate: " + result.candidateId;
    this.candName = "";
    this.candPhone = "";
    this.candIdNumber = "";
    this.candEducation = "";
    this.candExperienceYears = null;
    this.candSkills = "";
    await this.loadCandidates();
  }

  async searchCandidates() {
    const params: any = {};
    if (this.searchKeyword) params.keyword = this.searchKeyword;
    if (this.searchSkill) params.skill = this.searchSkill;
    if (this.searchEducation) params.education = this.searchEducation;
    this.searchResults = await this.api.searchCandidates(params);
  }
}