-- Generated SQL DDL
-- Database: mydb
-- Dialect: postgres

-- Search path: $user, public

CREATE TABLE "public"."approvals" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "issue_id" uuid NOT NULL,
  "approver_id" uuid NOT NULL,
  "status" character varying(50) NOT NULL,
  "comments" text,
  "approved_at" timestamp with time zone,
  "created_at" timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT "check_approval_status" CHECK (((status)::text = ANY ((ARRAY['Approved'::character varying, 'Rejected'::character varying, 'Pending'::character varying])::text[]))),
  CONSTRAINT "fk_approvals_approver" FOREIGN KEY ("approver_id") REFERENCES "public.users" ("id") ON DELETE RESTRICT ON UPDATE NO ACTION,
  CONSTRAINT "fk_approvals_issue" FOREIGN KEY ("issue_id") REFERENCES "public.issues" ("id") ON DELETE CASCADE ON UPDATE NO ACTION
);
CREATE INDEX "idx_approvals_approver" ON "public"."approvals" (approver_id);
CREATE INDEX "idx_approvals_created_at" ON "public"."approvals" (created_at DESC);
CREATE INDEX "idx_approvals_issue" ON "public"."approvals" (issue_id);
CREATE INDEX "idx_approvals_status" ON "public"."approvals" (status);
CREATE UNIQUE INDEX "unique_approval_per_user_issue" ON "public"."approvals" (issue_id, approver_id);

CREATE TABLE "public"."audit_logs" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "user_id" uuid NOT NULL,
  "action" character varying(100) NOT NULL,
  "resource_type" character varying(100) NOT NULL,
  "resource_id" uuid NOT NULL,
  "details" jsonb,
  "ip_address" inet,
  "user_agent" text,
  "timestamp" timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT "fk_audit_logs_user" FOREIGN KEY ("user_id") REFERENCES "public.users" ("id") ON DELETE RESTRICT ON UPDATE NO ACTION
);
CREATE INDEX "idx_audit_logs_action" ON "public"."audit_logs" (action);
CREATE INDEX "idx_audit_logs_details" ON "public"."audit_logs" USING GIN (details);
CREATE INDEX "idx_audit_logs_resource" ON "public"."audit_logs" (resource_type, resource_id);
CREATE INDEX "idx_audit_logs_timestamp" ON "public"."audit_logs" ("timestamp" DESC);
CREATE INDEX "idx_audit_logs_user" ON "public"."audit_logs" (user_id);

CREATE TABLE "public"."database_instances" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "name" character varying(100) NOT NULL,
  "description" text,
  "database_type" character varying(50) NOT NULL,
  "host" character varying(255) NOT NULL,
  "port" integer NOT NULL,
  "username" character varying(255) NOT NULL,
  "connection_details_encrypted" text,
  "use_rds_iam_auth" boolean DEFAULT false,
  "require_ssl" boolean DEFAULT true,
  "status" character varying(50) NOT NULL DEFAULT 'Active'::character varying,
  "created_at" timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "created_by_id" uuid NOT NULL,
  CONSTRAINT "check_database_type" CHECK (((database_type)::text = ANY ((ARRAY['mysql'::character varying, 'postgresql'::character varying])::text[]))),
  CONSTRAINT "check_instance_status" CHECK (((status)::text = ANY ((ARRAY['Active'::character varying, 'Inactive'::character varying, 'Maintenance'::character varying])::text[]))),
  CONSTRAINT "fk_instances_created_by" FOREIGN KEY ("created_by_id") REFERENCES "public.users" ("id") ON DELETE RESTRICT ON UPDATE NO ACTION
);
CREATE INDEX "idx_instances_created_by" ON "public"."database_instances" (created_by_id);
CREATE INDEX "idx_instances_name" ON "public"."database_instances" (name);
CREATE INDEX "idx_instances_status" ON "public"."database_instances" (status);
CREATE INDEX "idx_instances_type" ON "public"."database_instances" (database_type);
CREATE TRIGGER update_database_instances_updated_at BEFORE UPDATE ON public.database_instances FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


CREATE TABLE "public"."databases" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "project_id" uuid NOT NULL,
  "database_instance_id" uuid NOT NULL,
  "environment_id" uuid NOT NULL,
  "database_name" character varying(255) NOT NULL,
  "description" text,
  "status" character varying(50) NOT NULL DEFAULT 'Active'::character varying,
  "created_at" timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "created_by_id" uuid NOT NULL,
  CONSTRAINT "check_database_status" CHECK (((status)::text = ANY ((ARRAY['Active'::character varying, 'Inactive'::character varying])::text[]))),
  CONSTRAINT "fk_databases_created_by" FOREIGN KEY ("created_by_id") REFERENCES "public.users" ("id") ON DELETE RESTRICT ON UPDATE NO ACTION,
  CONSTRAINT "fk_databases_environment" FOREIGN KEY ("environment_id") REFERENCES "public.environments" ("id") ON DELETE RESTRICT ON UPDATE NO ACTION,
  CONSTRAINT "fk_databases_instance" FOREIGN KEY ("database_instance_id") REFERENCES "public.database_instances" ("id") ON DELETE RESTRICT ON UPDATE NO ACTION,
  CONSTRAINT "fk_databases_project" FOREIGN KEY ("project_id") REFERENCES "public.projects" ("id") ON DELETE CASCADE ON UPDATE NO ACTION
);
CREATE INDEX "idx_databases_created_by" ON "public"."databases" (created_by_id);
CREATE INDEX "idx_databases_environment" ON "public"."databases" (environment_id);
CREATE INDEX "idx_databases_instance" ON "public"."databases" (database_instance_id);
CREATE INDEX "idx_databases_project" ON "public"."databases" (project_id);
CREATE INDEX "idx_databases_status" ON "public"."databases" (status);
CREATE UNIQUE INDEX "unique_database_per_instance" ON "public"."databases" (database_instance_id, database_name);
CREATE TRIGGER update_databases_updated_at BEFORE UPDATE ON public.databases FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


CREATE TABLE "public"."environments" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "name" character varying(100) NOT NULL,
  "description" text,
  "status" character varying(50) NOT NULL DEFAULT 'Active'::character varying,
  "created_at" timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "created_by_id" uuid NOT NULL,
  CONSTRAINT "check_environment_status" CHECK (((status)::text = ANY ((ARRAY['Active'::character varying, 'Inactive'::character varying])::text[]))),
  CONSTRAINT "fk_environments_created_by" FOREIGN KEY ("created_by_id") REFERENCES "public.users" ("id") ON DELETE RESTRICT ON UPDATE NO ACTION
);
CREATE INDEX "idx_environments_created_by" ON "public"."environments" (created_by_id);
CREATE INDEX "idx_environments_name" ON "public"."environments" (name);
CREATE INDEX "idx_environments_status" ON "public"."environments" (status);
CREATE TRIGGER update_environments_updated_at BEFORE UPDATE ON public.environments FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


CREATE TABLE "public"."group_permissions" (
  "group_id" uuid NOT NULL,
  "permission_id" uuid NOT NULL,
  "granted_at" timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "granted_by_id" uuid NOT NULL,
  CONSTRAINT "fk_group_permissions_granted_by" FOREIGN KEY ("granted_by_id") REFERENCES "public.users" ("id") ON DELETE RESTRICT ON UPDATE NO ACTION,
  CONSTRAINT "fk_group_permissions_group" FOREIGN KEY ("group_id") REFERENCES "public.groups" ("id") ON DELETE CASCADE ON UPDATE NO ACTION,
  CONSTRAINT "fk_group_permissions_permission" FOREIGN KEY ("permission_id") REFERENCES "public.permissions" ("id") ON DELETE CASCADE ON UPDATE NO ACTION
);
CREATE INDEX "idx_group_permissions_group" ON "public"."group_permissions" (group_id);
CREATE INDEX "idx_group_permissions_permission" ON "public"."group_permissions" (permission_id);

CREATE TABLE "public"."groups" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "name" character varying(100) NOT NULL,
  "description" text,
  "created_at" timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX "groups_name_key" ON "public"."groups" (name);
CREATE INDEX "idx_groups_name" ON "public"."groups" (name);
CREATE TRIGGER update_groups_updated_at BEFORE UPDATE ON public.groups FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


CREATE TABLE "public"."issues" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "project_id" uuid NOT NULL,
  "target_database_id" uuid NOT NULL,
  "title" character varying(255) NOT NULL,
  "description" text NOT NULL,
  "change_type" character varying(50) NOT NULL,
  "sql_content" text NOT NULL,
  "rollback_sql" text,
  "status" character varying(50) NOT NULL DEFAULT 'Draft'::character varying,
  "risk_level" character varying(50) NOT NULL DEFAULT 'Medium'::character varying,
  "author_id" uuid NOT NULL,
  "assigned_reviewer_id" uuid,
  "scheduled_for" timestamp with time zone,
  "executed_at" timestamp with time zone,
  "execution_duration_ms" integer,
  "affected_rows" integer,
  "error_message" text,
  "created_at" timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT "check_change_type" CHECK (((change_type)::text = ANY ((ARRAY['DDL'::character varying, 'DML'::character varying])::text[]))),
  CONSTRAINT "check_issue_status" CHECK (((status)::text = ANY ((ARRAY['Draft'::character varying, 'PendingReview'::character varying, 'PendingApproval'::character varying, 'Approved'::character varying, 'Scheduled'::character varying, 'InProgress'::character varying, 'Completed'::character varying, 'Failed'::character varying, 'Cancelled'::character varying])::text[]))),
  CONSTRAINT "check_risk_level" CHECK (((risk_level)::text = ANY ((ARRAY['Low'::character varying, 'Medium'::character varying, 'High'::character varying, 'Critical'::character varying])::text[]))),
  CONSTRAINT "fk_issues_author" FOREIGN KEY ("author_id") REFERENCES "public.users" ("id") ON DELETE RESTRICT ON UPDATE NO ACTION,
  CONSTRAINT "fk_issues_database" FOREIGN KEY ("target_database_id") REFERENCES "public.databases" ("id") ON DELETE RESTRICT ON UPDATE NO ACTION,
  CONSTRAINT "fk_issues_project" FOREIGN KEY ("project_id") REFERENCES "public.projects" ("id") ON DELETE CASCADE ON UPDATE NO ACTION,
  CONSTRAINT "fk_issues_reviewer" FOREIGN KEY ("assigned_reviewer_id") REFERENCES "public.users" ("id") ON DELETE SET NULL ON UPDATE NO ACTION
);
CREATE INDEX "idx_issues_author" ON "public"."issues" (author_id);
CREATE INDEX "idx_issues_created_at" ON "public"."issues" (created_at DESC);
CREATE INDEX "idx_issues_database" ON "public"."issues" (target_database_id);
CREATE INDEX "idx_issues_project" ON "public"."issues" (project_id);
CREATE INDEX "idx_issues_reviewer" ON "public"."issues" (assigned_reviewer_id);
CREATE INDEX "idx_issues_risk_level" ON "public"."issues" (risk_level);
CREATE INDEX "idx_issues_scheduled" ON "public"."issues" (scheduled_for);
CREATE INDEX "idx_issues_status" ON "public"."issues" (status);
CREATE TRIGGER update_issues_updated_at BEFORE UPDATE ON public.issues FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


CREATE TABLE "public"."notifications" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "user_id" uuid NOT NULL,
  "notification_type" character varying(100) NOT NULL,
  "title" character varying(255) NOT NULL,
  "message" text NOT NULL,
  "related_resource_type" character varying(100),
  "related_resource_id" uuid,
  "is_read" boolean DEFAULT false,
  "created_at" timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "read_at" timestamp with time zone,
  CONSTRAINT "fk_notifications_user" FOREIGN KEY ("user_id") REFERENCES "public.users" ("id") ON DELETE CASCADE ON UPDATE NO ACTION
);
CREATE INDEX "idx_notifications_created_at" ON "public"."notifications" (created_at DESC);
CREATE INDEX "idx_notifications_unread" ON "public"."notifications" (user_id, is_read, created_at);
CREATE INDEX "idx_notifications_user" ON "public"."notifications" (user_id);

CREATE TABLE "public"."permissions" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "name" character varying(100) NOT NULL,
  "description" text,
  "resource_type" character varying(100) NOT NULL,
  "action" character varying(100) NOT NULL
);
CREATE INDEX "idx_permissions_resource" ON "public"."permissions" (resource_type, action);
CREATE UNIQUE INDEX "permissions_name_key" ON "public"."permissions" (name);
CREATE UNIQUE INDEX "unique_permission" ON "public"."permissions" (resource_type, action);

CREATE TABLE "public"."policy_violations" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "issue_id" uuid NOT NULL,
  "violation_type" character varying(100) NOT NULL,
  "severity" character varying(50) NOT NULL,
  "rule_name" character varying(255) NOT NULL,
  "message" text NOT NULL,
  "line_number" integer,
  "detected_at" timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT "check_severity" CHECK (((severity)::text = ANY ((ARRAY['Info'::character varying, 'Warning'::character varying, 'Error'::character varying, 'Critical'::character varying])::text[]))),
  CONSTRAINT "check_violation_type" CHECK (((violation_type)::text = ANY ((ARRAY['syntax_error'::character varying, 'security_risk'::character varying, 'performance_issue'::character varying, 'backwards_incompatible'::character varying, 'best_practice_violation'::character varying])::text[]))),
  CONSTRAINT "fk_violations_issue" FOREIGN KEY ("issue_id") REFERENCES "public.issues" ("id") ON DELETE CASCADE ON UPDATE NO ACTION
);
CREATE INDEX "idx_violations_issue" ON "public"."policy_violations" (issue_id);
CREATE INDEX "idx_violations_severity" ON "public"."policy_violations" (severity);

CREATE TABLE "public"."project_members" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "project_id" uuid NOT NULL,
  "user_id" uuid NOT NULL,
  "project_role" character varying(50) NOT NULL,
  "added_at" timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "added_by_id" uuid NOT NULL,
  CONSTRAINT "check_project_role" CHECK (((project_role)::text = ANY ((ARRAY['ProjectOwner'::character varying, 'ProjectDeveloper'::character varying, 'ProjectApprover'::character varying, 'ProjectViewer'::character varying])::text[]))),
  CONSTRAINT "fk_project_members_added_by" FOREIGN KEY ("added_by_id") REFERENCES "public.users" ("id") ON DELETE RESTRICT ON UPDATE NO ACTION,
  CONSTRAINT "fk_project_members_project" FOREIGN KEY ("project_id") REFERENCES "public.projects" ("id") ON DELETE CASCADE ON UPDATE NO ACTION,
  CONSTRAINT "fk_project_members_user" FOREIGN KEY ("user_id") REFERENCES "public.users" ("id") ON DELETE CASCADE ON UPDATE NO ACTION
);
CREATE INDEX "idx_project_members_project" ON "public"."project_members" (project_id);
CREATE INDEX "idx_project_members_role" ON "public"."project_members" (project_role);
CREATE INDEX "idx_project_members_user" ON "public"."project_members" (user_id);
CREATE UNIQUE INDEX "unique_project_user" ON "public"."project_members" (project_id, user_id);

CREATE TABLE "public"."projects" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "name" character varying(100) NOT NULL,
  "description" text,
  "status" character varying(50) NOT NULL DEFAULT 'Active'::character varying,
  "created_at" timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "created_by_id" uuid NOT NULL,
  CONSTRAINT "check_project_status" CHECK (((status)::text = ANY ((ARRAY['Active'::character varying, 'Archived'::character varying])::text[]))),
  CONSTRAINT "fk_projects_created_by" FOREIGN KEY ("created_by_id") REFERENCES "public.users" ("id") ON DELETE RESTRICT ON UPDATE NO ACTION
);
CREATE INDEX "idx_projects_created_by" ON "public"."projects" (created_by_id);
CREATE INDEX "idx_projects_name" ON "public"."projects" (name);
CREATE INDEX "idx_projects_status" ON "public"."projects" (status);
CREATE TRIGGER update_projects_updated_at BEFORE UPDATE ON public.projects FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


CREATE TABLE "public"."schema_drift_events" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "database_id" uuid NOT NULL,
  "snapshot_id" uuid NOT NULL,
  "drift_type" character varying(100) NOT NULL,
  "affected_object" character varying(255) NOT NULL,
  "drift_details" jsonb NOT NULL,
  "detected_at" timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "acknowledged_by_id" uuid,
  "acknowledged_at" timestamp with time zone,
  "resolution_notes" text,
  CONSTRAINT "check_drift_type" CHECK (((drift_type)::text = ANY ((ARRAY['table_added'::character varying, 'table_removed'::character varying, 'column_added'::character varying, 'column_removed'::character varying, 'column_modified'::character varying, 'index_added'::character varying, 'index_removed'::character varying, 'constraint_changed'::character varying])::text[]))),
  CONSTRAINT "fk_drift_acknowledged_by" FOREIGN KEY ("acknowledged_by_id") REFERENCES "public.users" ("id") ON DELETE SET NULL ON UPDATE NO ACTION,
  CONSTRAINT "fk_drift_database" FOREIGN KEY ("database_id") REFERENCES "public.databases" ("id") ON DELETE CASCADE ON UPDATE NO ACTION,
  CONSTRAINT "fk_drift_snapshot" FOREIGN KEY ("snapshot_id") REFERENCES "public.schema_snapshots" ("id") ON DELETE CASCADE ON UPDATE NO ACTION
);
CREATE INDEX "idx_drift_acknowledged" ON "public"."schema_drift_events" (acknowledged_at);
CREATE INDEX "idx_drift_database" ON "public"."schema_drift_events" (database_id);
CREATE INDEX "idx_drift_detected_at" ON "public"."schema_drift_events" (detected_at DESC);
CREATE INDEX "idx_drift_snapshot" ON "public"."schema_drift_events" (snapshot_id);

CREATE TABLE "public"."schema_migrations" (
  "version" bigint NOT NULL,
  "dirty" boolean NOT NULL
);

CREATE TABLE "public"."schema_snapshots" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "database_id" uuid NOT NULL,
  "schema_hash" character varying(64) NOT NULL,
  "schema_definition" jsonb NOT NULL,
  "captured_at" timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT "fk_snapshots_database" FOREIGN KEY ("database_id") REFERENCES "public.databases" ("id") ON DELETE CASCADE ON UPDATE NO ACTION
);
CREATE INDEX "idx_snapshots_captured_at" ON "public"."schema_snapshots" (captured_at DESC);
CREATE INDEX "idx_snapshots_database" ON "public"."schema_snapshots" (database_id);
CREATE INDEX "idx_snapshots_hash" ON "public"."schema_snapshots" (schema_hash);

CREATE TABLE "public"."user_groups" (
  "user_id" uuid NOT NULL,
  "group_id" uuid NOT NULL,
  "added_at" timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "added_by_id" uuid NOT NULL,
  CONSTRAINT "fk_user_groups_added_by" FOREIGN KEY ("added_by_id") REFERENCES "public.users" ("id") ON DELETE RESTRICT ON UPDATE NO ACTION,
  CONSTRAINT "fk_user_groups_group" FOREIGN KEY ("group_id") REFERENCES "public.groups" ("id") ON DELETE CASCADE ON UPDATE NO ACTION,
  CONSTRAINT "fk_user_groups_user" FOREIGN KEY ("user_id") REFERENCES "public.users" ("id") ON DELETE CASCADE ON UPDATE NO ACTION
);
CREATE INDEX "idx_user_groups_group" ON "public"."user_groups" (group_id);
CREATE INDEX "idx_user_groups_user" ON "public"."user_groups" (user_id);

CREATE TABLE "public"."users" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "email" character varying(255) NOT NULL,
  "password_hash" character varying(255),
  "display_name" character varying(255),
  "organization_role" character varying(50) NOT NULL DEFAULT 'OrganizationDeveloper'::character varying,
  "status" character varying(50) NOT NULL DEFAULT 'Active'::character varying,
  "oauth_provider" character varying(50),
  "oauth_id" character varying(255),
  "two_factor_enabled" boolean DEFAULT false,
  "two_factor_secret" character varying(255),
  "last_login_at" timestamp with time zone,
  "created_at" timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT "check_email_format" CHECK (((email)::text ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$'::text)),
  CONSTRAINT "check_organization_role" CHECK (((organization_role)::text = ANY ((ARRAY['OrganizationAdmin'::character varying, 'OrganizationDBA'::character varying, 'OrganizationDeveloper'::character varying])::text[]))),
  CONSTRAINT "check_status" CHECK (((status)::text = ANY ((ARRAY['Active'::character varying, 'Inactive'::character varying, 'Suspended'::character varying])::text[])))
);
CREATE INDEX "idx_users_email" ON "public"."users" (email);
CREATE INDEX "idx_users_oauth" ON "public"."users" (oauth_provider, oauth_id);
CREATE INDEX "idx_users_organization_role" ON "public"."users" (organization_role);
CREATE INDEX "idx_users_status" ON "public"."users" (status);
CREATE UNIQUE INDEX "users_email_key" ON "public"."users" (email);
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON public.users FOR EACH ROW EXECUTE FUNCTION public.update_updated_at_column();


CREATE OR REPLACE FUNCTION public.update_updated_at_column()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$function$
;

