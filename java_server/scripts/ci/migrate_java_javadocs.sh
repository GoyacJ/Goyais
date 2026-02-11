#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
cd "${REPO_ROOT}"

java_files=()
while IFS= read -r file; do
  [[ -n "${file}" ]] || continue
  java_files+=("${file}")
done < <(rg --files java_server -g '*.java' -g '!**/target/**' -g '!**/build/**')

perl - "${java_files[@]}" <<'PERL'
use strict;
use warnings;

sub trim {
  my ($s) = @_;
  $s //= q{};
  $s =~ s/^\s+|\s+$//g;
  return $s;
}

sub find_javadoc_bounds_before {
  my ($lines, $idx) = @_;
  my $cursor = $idx - 1;

  while ($cursor >= 0) {
    my $text = trim($lines->[$cursor]);
    if ($text eq q{} || $text =~ /^@/) {
      $cursor--;
      next;
    }
    last;
  }
  return if $cursor < 0;

  my $tail = trim($lines->[$cursor]);
  return if $tail !~ /\*\//;
  my $end = $cursor;

  while ($cursor >= 0) {
    my $text = trim($lines->[$cursor]);
    if ($text =~ /^\/\*\*/) {
      return ($cursor, $end);
    }
    if ($text =~ /^\/\*/ && $text !~ /^\/\*\*/) {
      return;
    }
    $cursor--;
  }

  return;
}

sub declaration_insert_line {
  my ($lines, $idx) = @_;
  my $insert = $idx;
  while ($insert > 0 && trim($lines->[$insert - 1]) =~ /^@/) {
    $insert--;
  }
  return $insert;
}

sub signature_from {
  my ($lines, $idx) = @_;
  my $sig = q{};
  my $seen_paren = 0;
  my $max = scalar(@{$lines}) - 1;
  my $limit = $idx + 60;
  $limit = $max if $limit > $max;

  for my $i ($idx .. $limit) {
    $sig .= q{ } . $lines->[$i];
    $seen_paren = 1 if $lines->[$i] =~ /\)/;
    if ($seen_paren && ($lines->[$i] =~ /\{/ || $lines->[$i] =~ /;/)) {
      last;
    }
  }

  $sig =~ s/\s+/ /g;
  $sig = trim($sig);
  return $sig;
}

sub extract_method_name {
  my ($signature) = @_;
  my $name = q{};
  while ($signature =~ /([A-Za-z_][A-Za-z0-9_]*)\s*\(/g) {
    $name = $1;
  }
  return $name;
}

sub split_top_level_commas {
  my ($text) = @_;
  my @parts;
  my $buf = q{};
  my $angle = 0;

  for my $ch (split //, $text) {
    if ($ch eq q{<}) {
      $angle++;
      $buf .= $ch;
      next;
    }
    if ($ch eq q{>}) {
      $angle-- if $angle > 0;
      $buf .= $ch;
      next;
    }
    if ($ch eq q{,} && $angle == 0) {
      push @parts, $buf;
      $buf = q{};
      next;
    }
    $buf .= $ch;
  }
  push @parts, $buf if $buf ne q{};

  return @parts;
}

sub extract_params {
  my ($signature) = @_;
  my $start = index($signature, q{(});
  return () if $start < 0;

  my $depth = 0;
  my $in = 0;
  my $param_text = q{};
  for (my $i = $start; $i < length($signature); $i++) {
    my $ch = substr($signature, $i, 1);
    if ($ch eq q{(}) {
      if ($in) {
        $depth++;
        $param_text .= $ch;
      } else {
        $in = 1;
      }
      next;
    }
    if ($ch eq q{)}) {
      if ($depth == 0) {
        last;
      }
      $depth--;
      $param_text .= $ch;
      next;
    }
    $param_text .= $ch if $in;
  }

  my @raw_parts = split_top_level_commas($param_text);
  my @params;
  for my $raw (@raw_parts) {
    my $p = trim($raw);
    next if $p eq q{};

    $p =~ s/\@\w+(?:\s*\([^()]*\))?\s*//g;
    $p =~ s/\b(final|volatile|transient)\b\s*//g;
    $p =~ s/\s+/ /g;
    $p = trim($p);
    next if $p eq q{};

    if ($p =~ /([A-Za-z_][A-Za-z0-9_]*)\s*(?:\[\s*\])*\s*$/) {
      push @params, $1;
    }
  }
  return @params;
}

sub extract_throws {
  my ($signature) = @_;
  return () if $signature !~ /\)\s*throws\s+([^\{;]+)/;

  my $throws_text = $1;
  my @raw = split_top_level_commas($throws_text);
  my @throws;
  for my $entry (@raw) {
    my $t = trim($entry);
    next if $t eq q{};
    $t =~ s/\s+/ /g;
    push @throws, $t;
  }

  return @throws;
}

sub has_tag {
  my ($doc, $pattern) = @_;
  return ($doc =~ /$pattern/m) ? 1 : 0;
}

sub add_javadoc_block {
  my ($lines, $insert_line, $indent, $summary, $tags_ref) = @_;
  my @block;
  push @block, "${indent}/**\n";
  push @block, "${indent} * <p>${summary}</p>\n";
  for my $tag (@{$tags_ref}) {
    push @block, "${indent} * ${tag}\n";
  }
  push @block, "${indent} */\n";
  splice @{$lines}, $insert_line, 0, @block;
}

sub append_missing_tags {
  my ($lines, $start, $end, $tags_ref) = @_;
  return 0 if !@{$tags_ref};

  my ($indent) = $lines->[$start] =~ /^(\s*)/;
  my @inserts = map { "${indent} * ${_}\n" } @{$tags_ref};
  splice @{$lines}, $end, 0, @inserts;
  return scalar(@inserts);
}

my @files = @ARGV;
my $updated_files = 0;

for my $file (@files) {
  open my $in, q{<}, $file or die "open($file): $!";
  my @lines = <$in>;
  close $in;
  my $changed = 0;

  for my $idx (0 .. $#lines) {
    if ($lines[$idx] =~ s/^(\s*\*\s*)\\@/$1@/) {
      $changed = 1;
    }
  }

  my %class_names;
  for my $line (@lines) {
    if ($line =~ /\b(?:class|interface|enum|record)\s+([A-Za-z_][A-Za-z0-9_]*)/) {
      $class_names{$1} = 1;
    }
  }

  my @decls;
  for my $i (0 .. $#lines) {
    my $line = $lines[$i];

    if ($line =~ /^\s*(public|protected)\s+(class|interface|enum|record)\b/) {
      push @decls, { kind => q{type}, line => $i };
      next;
    }

    if ($line =~ /^\s*(public|protected)\s+[^=;{}]*\(/ && $line !~ /\b(class|interface|enum|record)\b/) {
      push @decls, { kind => q{method}, line => $i };
      next;
    }

    if ($line =~ /^\s*(public|protected)\s+((?:static|final|transient|volatile|synchronized|strictfp)\s+)*[^=;(){}]+\s+[A-Za-z_][A-Za-z0-9_]*\s*(?:=|;)/) {
      push @decls, { kind => q{field}, line => $i };
      next;
    }
  }

  for my $decl (sort { $b->{line} <=> $a->{line} } @decls) {
    my $line_idx = $decl->{line};
    my $line = $lines[$line_idx];
    my ($indent) = $line =~ /^(\s*)/;

    my ($doc_start, $doc_end) = find_javadoc_bounds_before(\@lines, $line_idx);

    if (!defined $doc_start) {
      my @tags;
      if ($decl->{kind} eq q{method}) {
        my $sig = signature_from(\@lines, $line_idx);
        my $method_name = extract_method_name($sig);
        my @params = extract_params($sig);
        my @throws = extract_throws($sig);
        my $is_ctor = $class_names{$method_name} ? 1 : 0;
        my $is_void = $sig =~ /\bvoid\s+\Q$method_name\E\s*\(/ ? 1 : 0;

        for my $p (@params) {
          push @tags, "\@param ${p} TODO";
        }
        if (!$is_ctor && !$is_void) {
          push @tags, "\@return TODO";
        }
        for my $t (@throws) {
          my $simple = $t;
          $simple =~ s/^.*\.//;
          push @tags, "\@throws ${simple} TODO";
        }
      }

      my $insert_line = declaration_insert_line(\@lines, $line_idx);
      my $summary = $decl->{kind} eq q{type}
        ? q{TODO: describe type.}
        : $decl->{kind} eq q{field}
          ? q{TODO: describe field.}
          : q{TODO: describe method.};
      add_javadoc_block(\@lines, $insert_line, $indent, $summary, \@tags);
      $changed = 1;
      next;
    }

    if ($decl->{kind} ne q{method}) {
      next;
    }

    my $doc_text = join q{}, @lines[$doc_start .. $doc_end];
    if ($doc_text =~ /\{\@inheritDoc\}/) {
      next;
    }

    my $sig = signature_from(\@lines, $line_idx);
    my $method_name = extract_method_name($sig);
    next if $method_name eq q{};

    my @params = extract_params($sig);
    my @throws = extract_throws($sig);
    my $is_ctor = $class_names{$method_name} ? 1 : 0;
    my $is_void = $sig =~ /\bvoid\s+\Q$method_name\E\s*\(/ ? 1 : 0;

    my @missing_tags;
    for my $p (@params) {
      my $pattern = qr/\@param\s+\Q$p\E\b/;
      if (!has_tag($doc_text, $pattern)) {
        push @missing_tags, "\@param ${p} TODO";
      }
    }

    if (!$is_ctor && !$is_void && !has_tag($doc_text, qr/\@return\b/)) {
      push @missing_tags, '@return TODO';
    }

    for my $t (@throws) {
      my $simple = $t;
      $simple =~ s/^.*\.//;
      my $pattern = qr/\@throws\s+(\Q$t\E|\Q$simple\E)\b/;
      if (!has_tag($doc_text, $pattern)) {
        push @missing_tags, "\@throws ${simple} TODO";
      }
    }

    if (@missing_tags) {
      my $inserted = append_missing_tags(\@lines, $doc_start, $doc_end, \@missing_tags);
      $changed = 1 if $inserted > 0;
    }
  }

  if ($changed) {
    open my $out, q{>}, $file or die "write($file): $!";
    print {$out} @lines;
    close $out;
    $updated_files++;
  }
}

print "[migrate_java_javadocs] files_total=" . scalar(@files) . " files_updated=${updated_files}\n";
PERL
