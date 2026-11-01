#!/usr/bin/env node
import fs from 'node:fs/promises';
import path from 'node:path';

const BASE_URL = 'https://docs.bromcom.com/wp-json/wp/v2';
const OUTPUT_DIR = path.resolve(process.cwd(), 'user-stories');
const CONCURRENCY = 5;

function decodeHtmlEntities(str) {
  if (!str) return '';
  const named = {
    amp: '&',
    lt: '<',
    gt: '>',
    quot: '"',
    apos: "'",
    nbsp: ' ',
    ndash: '–',
    mdash: '—',
    lsquo: '‘',
    rsquo: '’',
    ldquo: '“',
    rdquo: '”',
    hellip: '…',
    pound: '£',
    copy: '©',
    reg: '®',
    bull: '•',
    rarr: '→',
    larr: '←',
  };
  return str
    .replace(/&#(\d+);/g, (_, dec) => String.fromCodePoint(parseInt(dec, 10)))
    .replace(/&#x([0-9a-fA-F]+);/g, (_, hex) => String.fromCodePoint(parseInt(hex, 16)))
    .replace(/&([a-zA-Z0-9]+);/g, (full, name) => (name in named ? named[name] : full));
}

function htmlToCompactMarkdown(html) {
  if (!html) return '';
  let s = html;

  // 1. Strip scripts, styles, SVGs, figures/images, audio/video/iframes
  s = s.replace(/<(script|style|svg|noscript|iframe|video|audio)\b[^>]*>[\s\S]*?<\/\1>/gi, '');
  s = s.replace(/<img\b[^>]*\/?>/gi, '');
  s = s.replace(/<figure\b[^>]*>\s*<\/figure>/gi, '');
  s = s.replace(/<figcaption\b[^>]*>([\s\S]*?)<\/figcaption>/gi, '\n*$1*\n');
  s = s.replace(/<\/?figure\b[^>]*>/gi, '');

  // 2. Convert headings
  s = s.replace(/<h1\b[^>]*>([\s\S]*?)<\/h1>/gi, '\n# $1\n');
  s = s.replace(/<h2\b[^>]*>([\s\S]*?)<\/h2>/gi, '\n## $1\n');
  s = s.replace(/<h3\b[^>]*>([\s\S]*?)<\/h3>/gi, '\n### $1\n');
  s = s.replace(/<h4\b[^>]*>([\s\S]*?)<\/h4>/gi, '\n#### $1\n');
  s = s.replace(/<h5\b[^>]*>([\s\S]*?)<\/h5>/gi, '\n##### $1\n');
  s = s.replace(/<h6\b[^>]*>([\s\S]*?)<\/h6>/gi, '\n###### $1\n');

  // 3. Convert tables to compact pipe-delimited rows
  s = s.replace(/<table\b[^>]*>([\s\S]*?)<\/table>/gi, (_, tableInner) => {
    const rows = [];
    const trMatches = tableInner.matchAll(/<tr\b[^>]*>([\s\S]*?)<\/tr>/gi);
    let isFirstRow = true;
    for (const tr of trMatches) {
      const cells = [];
      const cellMatches = tr[1].matchAll(/<(th|td)\b[^>]*>([\s\S]*?)<\/\1>/gi);
      for (const cell of cellMatches) {
        const cleanCell = decodeHtmlEntities(
          cell[2]
            .replace(/<[^>]+>/g, ' ')
            .replace(/\s+/g, ' ')
            .trim()
        ).replace(/\|/g, '\\|');
        cells.push(cleanCell);
      }
      if (cells.length > 0) {
        rows.push(`| ${cells.join(' | ')} |`);
        if (isFirstRow) {
          rows.push(`| ${cells.map(() => '---').join(' | ')} |`);
          isFirstRow = false;
        }
      }
    }
    return rows.length > 0 ? `\n${rows.join('\n')}\n` : '';
  });

  // 4. Convert lists
  s = s.replace(/<li\b[^>]*>([\s\S]*?)<\/li>/gi, (_, liContent) => {
    const cleaned = liContent.replace(/<\/?p\b[^>]*>/gi, ' ').trim();
    return `\n- ${cleaned}`;
  });
  s = s.replace(/<\/?(ul|ol)\b[^>]*>/gi, '\n');

  // 5. Convert inline formatting
  s = s.replace(/<(strong|b)\b[^>]*>([\s\S]*?)<\/\1>/gi, (_, _tag, inner) => {
    const trimmed = inner.trim();
    return trimmed ? `**${trimmed}**` : '';
  });
  s = s.replace(/<(em|i)\b[^>]*>([\s\S]*?)<\/\1>/gi, (_, _tag, inner) => {
    const trimmed = inner.trim();
    return trimmed ? `*${trimmed}*` : '';
  });
  s = s.replace(/<code\b[^>]*>([\s\S]*?)<\/code>/gi, '`$1`');
  s = s.replace(/<pre\b[^>]*>([\s\S]*?)<\/pre>/gi, '\n```\n$1\n```\n');

  // 6. Convert links
  s = s.replace(/<a\b[^>]*href=["']([^"']+)["'][^>]*>([\s\S]*?)<\/a>/gi, (_, href, text) => {
    const cleanText = text.replace(/<[^>]+>/g, '').trim();
    if (!cleanText) return '';
    if (href.startsWith('#') || href.startsWith('javascript:')) return cleanText;
    return `[${cleanText}](${href})`;
  });

  // 7. Convert block elements & line breaks
  s = s.replace(/<br\s*\/?>/gi, '\n');
  s = s.replace(/<hr\b[^>]*\/?>/gi, '\n---\n');
  s = s.replace(/<\/(p|div|section|article|blockquote|header|footer|aside)>/gi, '\n\n');
  s = s.replace(/<(blockquote)\b[^>]*>/gi, '\n> ');

  // 8. Strip all remaining HTML tags
  s = s.replace(/<[^>]+>/g, '');

  // 9. Decode HTML entities
  s = decodeHtmlEntities(s);

  // 10. Clean up redundant whitespace & blank lines
  s = s
    .replace(/\u00a0/g, ' ')
    .replace(/[ \t]+\n/g, '\n')
    .replace(/\n[ \t]+/g, '\n')
    .replace(/[ \t]{2,}/g, ' ')
    .replace(/\*\* \*\*/g, ' ')
    .replace(/\*\*\*\*/g, '')
    .replace(/\n{3,}/g, '\n\n')
    .trim();

  return s;
}

async function fetchJsonWithHeaders(url, retries = 3) {
  for (let attempt = 1; attempt <= retries; attempt++) {
    try {
      const res = await fetch(url, {
        headers: {
          'User-Agent': 'Mozilla/5.0 (X11; Linux x86_64) OptimisDocsExtractor/1.0',
          Accept: 'application/json',
        },
      });
      if (!res.ok) {
        throw new Error(`HTTP ${res.status} for ${url}`);
      }
      const totalPages = parseInt(res.headers.get('x-wp-totalpages') || '1', 10);
      const totalItems = parseInt(res.headers.get('x-wp-total') || '0', 10);
      const data = await res.json();
      return { data, totalPages, totalItems };
    } catch (err) {
      if (attempt === retries) throw err;
      await new Promise((r) => setTimeout(r, 1000 * attempt));
    }
  }
}

async function fetchAllPaginated(endpoint, queryParams = '') {
  const perPage = 100;
  const sep = queryParams ? '&' : '';
  const firstUrl = `${BASE_URL}/${endpoint}?per_page=${perPage}${sep}${queryParams}`;
  const first = await fetchJsonWithHeaders(firstUrl);
  const all = [...first.data];
  const totalPages = first.totalPages;
  console.log(`[${endpoint}] Total items: ${first.totalItems}, pages: ${totalPages}`);

  const pages = [];
  for (let p = 2; p <= totalPages; p++) {
    pages.push(p);
  }

  for (let i = 0; i < pages.length; i += CONCURRENCY) {
    const batch = pages.slice(i, i + CONCURRENCY);
    const results = await Promise.all(
      batch.map(async (page) => {
        const url = `${BASE_URL}/${endpoint}?per_page=${perPage}&page=${page}${sep}${queryParams}`;
        const res = await fetchJsonWithHeaders(url);
        console.log(`  Fetched ${endpoint} page ${page}/${totalPages} (${res.data.length} items)`);
        return res.data;
      })
    );
    for (const items of results) {
      all.push(...items);
    }
  }

  return all;
}

function sanitizeSlug(slug) {
  return (
    (slug || 'untitled')
      .toLowerCase()
      .replace(/[^a-z0-9-_]+/g, '-')
      .replace(/^-+|-+$/g, '')
      .slice(0, 80) || 'untitled'
  );
}

// Classify uncategorized articles by title/slug/tags/body keywords so every article lands in the most relevant domain folder
function inferCategorySlug(title, slug, tagsText, body) {
  const hay = `${title} ${slug} ${tagsText}`.toLowerCase();
  const full = `${hay} ${body.slice(0, 600).toLowerCase()}`;

  if (/\b(census|plasc|swfc?|swf|collect|post[- ]?16|eyfs profile return|s2s|ctf|atf|uln|upn)\b/.test(hay)) {
    return 'census';
  }
  if (/\b(safeguard|bullying|child protection|cpoms|prevent|looked after|lac)\b/.test(hay)) {
    return 'safeguarding';
  }
  if (/\b(attendance|absence|register|truancy|missing register|subcode|late mark|detention.*register)\b/.test(hay)) {
    return 'attendance';
  }
  if (/\b(behaviour|behavior|detention|on[- ]?report|exclusion|suspension|reward|merit|house point|incident|event alert|referral)\b/.test(hay)) {
    return 'behaviour';
  }
  if (/\b(exam|examinations|jcq|basedata|candidate|invigilat|mock exam|seating|edi|uci|entries|embargo|btec|gcse|a[- ]?level|qn data)\b/.test(hay)) {
    return 'examinations';
  }
  if (/\b(finance|invoice|vat|ledger|cost centre|bacs|cheque|accounts receivable|accounts payable|purchase order|reconcil|virement|journal|budget|cfr return|salary|salaries|supplier|paypoint|refund|commitment|sundry)\b/.test(hay)) {
    return 'finance';
  }
  if (/\b(mcas|mychildatschool|my child at school|parent portal|parent app|parents evening|parental consent|admission form|data collection form|club|trip|wraparound)\b/.test(hay)) {
    return 'mychildatschool';
  }
  if (/\b(dinner|meal|catering|fsm|free school meal|breakfast club)\b/.test(hay)) {
    return 'dinner';
  }
  if (/\b(assessment|marksheet|grade set|mark set|ks2|ks4|ks5|primary tracker|tracker|attainment|progress 8|target|cats|baseline)\b/.test(hay)) {
    return 'assessment';
  }
  if (/\b(timetable|e-?timetable|nova-?t|edval|timetabler|period structure|cycle|academic day|curriculum plan|options|band|block|room booking|room closure)\b/.test(hay)) {
    return 'timetable-and-curriculum';
  }
  if (/\b(cover|supply agency|supply staff)\b/.test(hay)) {
    return 'cover';
  }
  if (/\b(teacher app)\b/.test(hay)) {
    return 'for-teachers';
  }
  if (/\b(student portal|homework)\b/.test(hay)) {
    return 'student-portal';
  }
  if (/\b(vision|power bi|mat |trust|centralised|distribute)\b/.test(hay)) {
    return 'vision-and-mat';
  }
  if (/\b(sen|send|senco|ehcp|special educational|access arrangement|medical|doctor|health)\b/.test(hay)) {
    return 'sen-and-medical';
  }
  if (/\b(sms|email|communication|message template|announcement|notification|watch ?list|smtp)\b/.test(hay)) {
    return 'communications';
  }
  if (/\b(staff|scr|single central record|cpd|performance management|line management|contract|leaver|recruitment|governor)\b/.test(hay)) {
    return 'staff-and-hr';
  }
  if (/\b(report|adhoc|mail merge|matrix report|quick report|odata|excel add-in|google sheet|export|import)\b/.test(hay)) {
    return 'reporting-and-data';
  }
  if (/\b(group|tutor group|teaching class|house|faculty|department|year group|withdrawal group)\b/.test(hay)) {
    return 'groups';
  }
  if (/\b(student|pupil|applicant|admission|leaver|enrolment|contact|young carer|photo| sibling|family)\b/.test(hay)) {
    return 'students-and-admissions';
  }
  if (/\b(role|permission|sso|single sign|password|ldap|office ?365|google integration|setup|configuration|lookup|user defined|audit|gdpr|sar|archive|retention|cache|dms)\b/.test(full)) {
    return 'setup-and-administration';
  }
  return 'general-and-support';
}

// Normalize WordPress top-level category slugs into clean domain folder names
function mapWpCategoryToFolder(topSlug, fullCatPath, title, slug, tagsText, body) {
  switch (topSlug) {
    case 'census':
      return 'census';
    case 'finance':
      return 'finance';
    case 'examinations':
      return 'examinations';
    case 'mychildatschool':
    case 'for-parents':
      return 'mychildatschool';
    case 'dinner':
      return 'dinner';
    case 'assessment':
    case 'analysis':
      return 'assessment';
    case 'behaviour':
      return 'behaviour';
    case 'cover':
      return 'cover';
    case 'curriculum':
    case 'timetable':
      return 'timetable-and-curriculum';
    case 'for-teachers':
      return 'for-teachers';
    case 'for-students':
    case 'student-portal':
      return 'student-portal';
    case 'groups':
      return 'groups';
    case 'reporting':
      return 'reporting-and-data';
    case 'staff':
      return 'staff-and-hr';
    case 'student':
      return 'students-and-admissions';
    case 'setup':
    case 'administration':
      return 'setup-and-administration';
    case 'vision-power-bi':
    case 'virtual-school':
      return 'vision-and-mat';
    default:
      return inferCategorySlug(title, slug, `${fullCatPath} ${tagsText}`, body);
  }
}

async function main() {
  console.log(`Starting extraction & folder categorisation from ${BASE_URL} into ${OUTPUT_DIR}...`);

  // Clean existing files in user-stories/ first so we have a clean folder structure
  await fs.rm(OUTPUT_DIR, { recursive: true, force: true });
  await fs.mkdir(OUTPUT_DIR, { recursive: true });

  // 1. Fetch categories
  const categoriesRaw = await fetchAllPaginated('ht-kb-category');
  const categoryMap = new Map();
  for (const cat of categoriesRaw) {
    categoryMap.set(cat.id, {
      id: cat.id,
      name: decodeHtmlEntities(cat.name),
      slug: cat.slug,
      parent: cat.parent || 0,
    });
  }

  function getTopCategorySlug(catIds) {
    if (!catIds || catIds.length === 0) return 'uncategorized';
    let current = categoryMap.get(catIds[0]);
    if (!current) return 'uncategorized';
    const visited = new Set();
    while (current.parent && categoryMap.has(current.parent) && !visited.has(current.id)) {
      visited.add(current.id);
      current = categoryMap.get(current.parent);
    }
    return sanitizeSlug(current.slug);
  }

  function getCategoryPathNames(catIds) {
    if (!catIds || catIds.length === 0) return [];
    return catIds
      .map((id) => {
        const chain = [];
        let cur = categoryMap.get(id);
        const visited = new Set();
        while (cur && !visited.has(cur.id)) {
          visited.add(cur.id);
          chain.unshift(cur.name);
          cur = cur.parent ? categoryMap.get(cur.parent) : null;
        }
        return chain.join(' > ');
      })
      .filter(Boolean);
  }

  // 2. Fetch all KB articles (ht-kb), pages, and posts
  const kbArticles = await fetchAllPaginated(
    'ht-kb',
    '_fields=id,slug,title,content,link,modified,ht-kb-category,tpgb_post_category'
  );
  const pages = await fetchAllPaginated('pages', '_fields=id,slug,title,content,link,modified');
  const posts = await fetchAllPaginated('posts', '_fields=id,slug,title,content,link,modified');

  const folders = new Map();
  const indexEntries = [];

  for (const item of kbArticles) {
    const title = decodeHtmlEntities(item.title?.rendered || '').trim();
    const body = htmlToCompactMarkdown(item.content?.rendered || '');
    if (!body && !title) continue;

    const catIds = item['ht-kb-category'] || [];
    const topSlug = getTopCategorySlug(catIds);
    const catPaths = getCategoryPathNames(catIds);
    const tagsRaw = (item.tpgb_post_category?.ht_kb_tag || '').replace(/<[^>]+>/g, ' ').trim();

    const folderName = mapWpCategoryToFolder(
      topSlug,
      catPaths.join(' '),
      title,
      item.slug,
      tagsRaw,
      body
    );

    if (!folders.has(folderName)) {
      folders.set(folderName, []);
    }

    const record = {
      id: item.id,
      slug: sanitizeSlug(item.slug || `${item.id}`),
      title,
      url: item.link,
      modified: (item.modified || '').slice(0, 10),
      categories: catPaths.length ? catPaths : [folderName],
      body,
    };
    folders.get(folderName).push(record);
    indexEntries.push({
      id: record.id,
      slug: record.slug,
      title: record.title,
      folder: folderName,
      categories: record.categories,
      url: record.url,
    });
  }

  for (const item of [...pages, ...posts]) {
    const title = decodeHtmlEntities(item.title?.rendered || '').trim();
    const body = htmlToCompactMarkdown(item.content?.rendered || '');
    if (!body || body.length < 40) continue;

    const folderName = inferCategorySlug(title, item.slug, '', body);
    if (!folders.has(folderName)) {
      folders.set(folderName, []);
    }

    const record = {
      id: item.id,
      slug: sanitizeSlug(item.slug || `${item.id}`),
      title,
      url: item.link,
      modified: (item.modified || '').slice(0, 10),
      categories: ['Pages'],
      body,
    };
    folders.get(folderName).push(record);
    indexEntries.push({
      id: record.id,
      slug: record.slug,
      title: record.title,
      folder: folderName,
      categories: record.categories,
      url: record.url,
    });
  }

  // 3. Write individual space-efficient article files inside each category folder
  let totalBytes = 0;
  let totalFiles = 0;
  const folderStats = [];

  for (const [folderName, articles] of [...folders.entries()].sort((a, b) => a[0].localeCompare(b[0]))) {
    const folderPath = path.join(OUTPUT_DIR, folderName);
    await fs.mkdir(folderPath, { recursive: true });

    articles.sort((a, b) => a.slug.localeCompare(b.slug));
    const usedSlugs = new Set();
    let folderBytes = 0;

    for (const art of articles) {
      let fileSlug = art.slug;
      if (usedSlugs.has(fileSlug)) {
        fileSlug = `${fileSlug}-${art.id}`;
      }
      usedSlugs.add(fileSlug);
      art.fileName = `${fileSlug}.md`;

      const catHeader = art.categories.length ? `Categories: ${art.categories.join(' | ')}\n` : '';
      const content = `# ${art.title}\nID: ${art.id} | Updated: ${art.modified}\n${catHeader}\n${art.body}\n`;
      const bytes = Buffer.byteLength(content, 'utf8');

      await fs.writeFile(path.join(folderPath, art.fileName), content, 'utf8');
      folderBytes += bytes;
      totalBytes += bytes;
      totalFiles++;
    }

    // Write a compact INDEX.md inside each category folder
    const folderIndexLines = [
      `# ${folderName} (${articles.length} articles)`,
      '',
      ...articles.map((a) => `- [${a.title}](./${a.fileName}) (ID: ${a.id}, Updated: ${a.modified})`),
    ];
    await fs.writeFile(path.join(folderPath, 'INDEX.md'), folderIndexLines.join('\n') + '\n', 'utf8');

    folderStats.push({
      folderName,
      count: articles.length,
      bytes: folderBytes,
    });
  }

  // 4. Write master INDEX.md in user-stories/
  const masterLines = [
    '# Bromcom Documentation Extracted Index',
    `Total Articles: ${totalFiles} | Category Folders: ${folderStats.length} | Total Size: ${(
      totalBytes /
      (1024 * 1024)
    ).toFixed(2)} MB`,
    '',
    '## Category Folders',
    ...folderStats.map(
      (s) =>
        `- [**${s.folderName}/**](./${s.folderName}/INDEX.md): ${s.count} articles (${(
          s.bytes / 1024
        ).toFixed(1)} KB)`
    ),
  ];
  await fs.writeFile(path.join(OUTPUT_DIR, 'INDEX.md'), masterLines.join('\n') + '\n', 'utf8');

  console.log(
    `Done! Extracted ${totalFiles} articles into ${folderStats.length} category folders (${(
      totalBytes /
      (1024 * 1024)
    ).toFixed(2)} MB total).`
  );
}

main().catch((err) => {
  console.error('Extraction failed:', err);
  process.exit(1);
});
