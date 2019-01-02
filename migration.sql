CREATE TYPE "production"."review_status" AS ENUM (
	'waiting',
	'accepted',
	'rejected'
);

ALTER TABLE "production"."productreview" ADD COLUMN "status" "production"."review_status" DEFAULT 'waiting'::"production"."review_status" NOT NULL;
