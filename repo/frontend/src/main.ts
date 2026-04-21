import { bootstrapApplication } from "@angular/platform-browser";
import { provideRouter, Routes } from "@angular/router";
import { provideHttpClient, withInterceptorsFromDi } from "@angular/common/http";
import { AppComponent } from "./app/app.component";
import { RecruitmentComponent } from "./app/recruitment/recruitment";
import { ComplianceComponent } from "./app/compliance/compliance";
import { CaseledgerComponent } from "./app/caseledger/caseledger";
import { AuditComponent } from "./app/audit/audit";
import { TagsComponent } from "./app/tags/tags";

const routes: Routes = [
  { path: 'recruitment', component: RecruitmentComponent },
  { path: 'compliance', component: ComplianceComponent },
  { path: 'cases', component: CaseledgerComponent },
  { path: 'audit', component: AuditComponent },
  { path: 'tags', component: TagsComponent },
  { path: '', redirectTo: '/recruitment', pathMatch: 'full' }
];

bootstrapApplication(AppComponent, {
  providers: [
    provideRouter(routes),
    provideHttpClient(withInterceptorsFromDi())
  ]
}).catch((err) => console.error(err));