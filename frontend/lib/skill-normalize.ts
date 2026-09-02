/** Shared skill normalization for resume parsing review and match score. */

const SKILL_ALIASES: Record<string, string> = {
  javascript: "JavaScript",
  "java script": "JavaScript",
  js: "JavaScript",
  ecmascript: "JavaScript",
  typescript: "TypeScript",
  ts: "TypeScript",
  react: "React",
  "react.js": "React",
  reactjs: "React",
  "react js": "React",
  node: "Node.js",
  nodejs: "Node.js",
  "node.js": "Node.js",
  "node js": "Node.js",
  next: "Next.js",
  nextjs: "Next.js",
  "next.js": "Next.js",
  "next js": "Next.js",
  mongodb: "MongoDB",
  mongo: "MongoDB",
  "mongo db": "MongoDB",
  postgresql: "PostgreSQL",
  postgres: "PostgreSQL",
  mysql: "MySQL",
  golang: "Go",
  go: "Go",
  python: "Python",
  java: "Java",
  docker: "Docker",
  kubernetes: "Kubernetes",
  k8s: "Kubernetes",
  aws: "AWS",
  azure: "Azure",
  gcp: "GCP",
  "google cloud": "GCP",
  html: "HTML",
  css: "CSS",
  tailwind: "Tailwind",
  vue: "Vue",
  "vue.js": "Vue",
  angular: "Angular",
  redux: "Redux",
  graphql: "GraphQL",
  rest: "REST",
  git: "Git",
  linux: "Linux",
  redis: "Redis",
  express: "Express",
  "express.js": "Express",
};

function compactKey(skill: string): string {
  return skill
    .toLowerCase()
    .trim()
    .replace(/[-_]/g, " ")
    .replace(/\./g, " ")
    .replace(/\s+/g, " ")
    .trim();
}

export function normalizeSkillKey(skill: string): string {
  const spaced = compactKey(skill);
  const nospace = spaced.replace(/\s+/g, "");
  const canon =
    SKILL_ALIASES[spaced] ||
    SKILL_ALIASES[nospace] ||
    SKILL_ALIASES[spaced.replace(/\.js$/, "")] ||
    null;
  if (canon) {
    return compactKey(canon).replace(/\s+/g, "");
  }
  return nospace;
}

export function canonicalSkillName(skill: string): string {
  const spaced = compactKey(skill);
  const nospace = spaced.replace(/\s+/g, "");
  return (
    SKILL_ALIASES[spaced] ||
    SKILL_ALIASES[nospace] ||
    skill.trim()
  );
}

export function dedupeSkills(skills: string[]): string[] {
  const seen = new Set<string>();
  const out: string[] = [];
  for (const skill of skills) {
    const trimmed = skill.trim();
    if (!trimmed) continue;
    const key = normalizeSkillKey(trimmed);
    if (seen.has(key)) continue;
    seen.add(key);
    out.push(canonicalSkillName(trimmed));
  }
  return out;
}

export function skillsMatch(required: string, candidateSkills: string[]): boolean {
  const reqKey = normalizeSkillKey(required);
  return candidateSkills.some((s) => normalizeSkillKey(s) === reqKey);
}
