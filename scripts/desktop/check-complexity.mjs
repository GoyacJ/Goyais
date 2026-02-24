#!/usr/bin/env node
import fs from "node:fs";
import { createRequire } from "node:module";
import path from "node:path";

const repoRoot = path.resolve(path.dirname(new URL(import.meta.url).pathname), "..", "..");
const desktopSrcPath = path.join(repoRoot, "apps/desktop/src");
const desktopPackagePath = path.join(repoRoot, "apps/desktop");
const requireFromScript = createRequire(import.meta.url);
const ts = requireFromScript(requireFromScript.resolve("typescript", { paths: [desktopPackagePath] }));

const cyclomaticThreshold = 10;
const cognitiveThreshold = 15;
const sourceFileExtensions = new Set([".ts", ".tsx", ".vue"]);
const ignoredFilePatterns = [/\.spec\.[cm]?[jt]sx?$/i, /\/tests\//, /\.d\.ts$/i];

function main() {
  if (!fs.existsSync(desktopSrcPath)) {
    fail([`desktop src path not found: ${desktopSrcPath}`]);
  }

  const files = walkFiles(desktopSrcPath).filter((filePath) => {
    const ext = path.extname(filePath).toLowerCase();
    if (!sourceFileExtensions.has(ext)) {
      return false;
    }
    const normalized = filePath.replaceAll(path.sep, "/");
    return !ignoredFilePatterns.some((pattern) => pattern.test(normalized));
  });

  const violations = [];
  for (const filePath of files) {
    const sourceUnits = parseSourceUnits(filePath);
    for (const unit of sourceUnits) {
      const sourceFile = ts.createSourceFile(
        unit.virtualFilePath,
        unit.content,
        ts.ScriptTarget.Latest,
        true,
        ts.ScriptKind.TS
      );
      visitFunctionLikeNodes(sourceFile, (node) => {
        const metrics = computeComplexity(node);
        if (metrics.cyclomatic <= cyclomaticThreshold && metrics.cognitive <= cognitiveThreshold) {
          return;
        }
        const position = sourceFile.getLineAndCharacterOfPosition(node.getStart(sourceFile));
        const functionName = resolveFunctionName(node);
        violations.push({
          filePath,
          line: unit.lineOffset + position.line + 1,
          functionName,
          cyclomatic: metrics.cyclomatic,
          cognitive: metrics.cognitive
        });
      });
    }
  }

  if (violations.length > 0) {
    const lines = violations
      .sort((a, b) => (b.cognitive === a.cognitive ? b.cyclomatic - a.cyclomatic : b.cognitive - a.cognitive))
      .map((item) => {
        const relativePath = path.relative(repoRoot, item.filePath);
        return `${relativePath}:${item.line} ${item.functionName} => cyclomatic ${item.cyclomatic} (<=${cyclomaticThreshold}), cognitive ${item.cognitive} (<=${cognitiveThreshold})`;
      });
    fail(lines);
  }

  console.log("[check-complexity] OK");
  console.log(`- scanned: ${files.length} files`);
}

function walkFiles(startPath) {
  const files = [];
  const stack = [startPath];
  while (stack.length > 0) {
    const currentPath = stack.pop();
    if (!currentPath) {
      continue;
    }
    const stat = fs.statSync(currentPath);
    if (stat.isDirectory()) {
      const entries = fs.readdirSync(currentPath, { withFileTypes: true });
      for (const entry of entries) {
        stack.push(path.join(currentPath, entry.name));
      }
      continue;
    }
    files.push(currentPath);
  }
  return files;
}

function parseSourceUnits(filePath) {
  const ext = path.extname(filePath).toLowerCase();
  const sourceText = fs.readFileSync(filePath, "utf8");
  if (ext === ".vue") {
    const units = [];
    const scriptPattern = /<script\b[^>]*>([\s\S]*?)<\/script>/gi;
    let match = scriptPattern.exec(sourceText);
    while (match) {
      const content = match[1] ?? "";
      const fullMatch = match[0] ?? "";
      const contentIndex = (match.index ?? 0) + Math.max(fullMatch.indexOf(content), 0);
      const lineOffset = sourceText.slice(0, contentIndex).split(/\r?\n/).length - 1;
      units.push({
        virtualFilePath: `${filePath}#script`,
        content,
        lineOffset
      });
      match = scriptPattern.exec(sourceText);
    }
    return units;
  }

  return [
    {
      virtualFilePath: filePath,
      content: sourceText,
      lineOffset: 0
    }
  ];
}

function visitFunctionLikeNodes(node, onFunctionLike) {
  const visit = (currentNode) => {
    if (ts.isFunctionLike(currentNode)) {
      if (currentNode.body) {
        onFunctionLike(currentNode);
      }
      return;
    }
    ts.forEachChild(currentNode, visit);
  };
  visit(node);
}

function computeComplexity(functionNode) {
  let cyclomatic = 1;
  let cognitive = 0;

  const rootNode = functionNode.body ?? functionNode;

  const visit = (node, nestingLevel) => {
    if (node !== rootNode && ts.isFunctionLike(node)) {
      return;
    }

    if (ts.isIfStatement(node)) {
      cyclomatic += 1;
      cognitive += 1 + nestingLevel;
      visit(node.expression, nestingLevel);
      visit(node.thenStatement, nestingLevel + 1);
      if (node.elseStatement) {
        if (ts.isIfStatement(node.elseStatement)) {
          visit(node.elseStatement, nestingLevel);
        } else {
          visit(node.elseStatement, nestingLevel + 1);
        }
      }
      return;
    }

    if (ts.isForStatement(node) || ts.isForInStatement(node) || ts.isForOfStatement(node) || ts.isWhileStatement(node) || ts.isDoStatement(node)) {
      cyclomatic += 1;
      cognitive += 1 + nestingLevel;
      ts.forEachChild(node, (child) => {
        const nextNesting = child === node.statement ? nestingLevel + 1 : nestingLevel;
        visit(child, nextNesting);
      });
      return;
    }

    if (ts.isSwitchStatement(node)) {
      cyclomatic += 1;
      cognitive += 1 + nestingLevel;
      visit(node.expression, nestingLevel);
      for (const clause of node.caseBlock.clauses) {
        visit(clause, nestingLevel + 1);
      }
      return;
    }

    if (ts.isCaseClause(node)) {
      cyclomatic += 1;
      cognitive += 1 + nestingLevel;
      for (const statement of node.statements) {
        visit(statement, nestingLevel + 1);
      }
      return;
    }

    if (ts.isConditionalExpression(node)) {
      cyclomatic += 1;
      cognitive += 1 + nestingLevel;
      visit(node.condition, nestingLevel);
      visit(node.whenTrue, nestingLevel + 1);
      visit(node.whenFalse, nestingLevel + 1);
      return;
    }

    if (ts.isCatchClause(node)) {
      cyclomatic += 1;
      cognitive += 1 + nestingLevel;
      visit(node.block, nestingLevel + 1);
      return;
    }

    if (ts.isBinaryExpression(node)) {
      const operatorKind = node.operatorToken.kind;
      if (
        operatorKind === ts.SyntaxKind.AmpersandAmpersandToken
        || operatorKind === ts.SyntaxKind.BarBarToken
        || operatorKind === ts.SyntaxKind.QuestionQuestionToken
      ) {
        cyclomatic += 1;
        cognitive += 1 + nestingLevel;
      }
    }

    ts.forEachChild(node, (child) => visit(child, nestingLevel));
  };

  visit(rootNode, 0);
  return { cyclomatic, cognitive };
}

function resolveFunctionName(node) {
  if (node.name && ts.isIdentifier(node.name)) {
    return node.name.text;
  }
  const parentNode = node.parent;
  if (parentNode && ts.isVariableDeclaration(parentNode) && ts.isIdentifier(parentNode.name)) {
    return parentNode.name.text;
  }
  if (parentNode && ts.isPropertyAssignment(parentNode) && ts.isIdentifier(parentNode.name)) {
    return parentNode.name.text;
  }
  if (parentNode && ts.isMethodDeclaration(parentNode) && parentNode.name && ts.isIdentifier(parentNode.name)) {
    return parentNode.name.text;
  }
  return "<anonymous>";
}

function fail(lines) {
  console.error("[check-complexity] FAILED");
  for (const line of lines) {
    console.error(`- ${line}`);
  }
  process.exit(1);
}

main();
