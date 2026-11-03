#!/usr/bin/env bash
# The severity gate of a release: fails on a critical CodeQL alert that the release analysis reports as
# open, and warns about high ones. An alert also has instances from the weekly scheduled analysis, which
# can stay open after a fix until its next run, so only the release analysis decides.
set -euo pipefail
# A failed read inside a command substitution has to stop the gate too, not read as "no alerts".
shopt -s inherit_errexit

repository=$1
ref=$2
commit=$3

release_analysis_key=".github/workflows/ci-release.yml:analyze"
categories=(/language:go /language:javascript-typescript /language:actions)

# The newest page is enough: the analysis of the released commit was uploaded moments ago. The key of a
# called workflow is built by CodeQL, so a wrong one has to stop the release instead of matching no alert.
analyses=$(gh api "repos/$repository/code-scanning/analyses?ref=$ref&tool_name=CodeQL&sort=created&direction=desc&per_page=100")

for category in "${categories[@]}"; do
    release_analysis_count=$(jq --arg key "$release_analysis_key" --arg commit "$commit" --arg category "$category" \
        '[.[] | select(.analysis_key == $key and .commit_sha == $commit and .category == $category)] | length' \
        <<<"$analyses")

    if [ "$release_analysis_count" -eq 0 ]; then
        echo "no release analysis of $commit under $release_analysis_key for $category"
        exit 1
    fi
done

list_alerts_open_in_release_analysis() {
    local severity=$1
    local alerts
    local alert_number
    local alert_description
    local instances
    local is_open_in_release_analysis

    alerts=$(gh api --paginate "repos/$repository/code-scanning/alerts?ref=$ref&state=open&tool_name=CodeQL&severity=$severity&per_page=100" |
        jq -c -s 'add // [] | .[] | {number, description: "\(.rule.id) \(.html_url)"}')

    while IFS= read -r alert; do
        if [ -z "$alert" ]; then
            continue
        fi

        alert_number=$(jq -r '.number' <<<"$alert")
        alert_description=$(jq -r '.description' <<<"$alert")

        instances=$(gh api --paginate "repos/$repository/code-scanning/alerts/$alert_number/instances?ref=$ref&per_page=100")

        is_open_in_release_analysis=$(jq -s --arg key "$release_analysis_key" \
            'add // [] | any(.analysis_key == $key and .state == "open")' <<<"$instances")

        if [ "$is_open_in_release_analysis" = "true" ]; then
            echo "$alert_description"
        fi
    done <<<"$alerts"
}

critical_alerts=$(list_alerts_open_in_release_analysis critical)
high_alerts=$(list_alerts_open_in_release_analysis high)

while IFS= read -r high_alert; do
    if [ -n "$high_alert" ]; then
        echo "::warning::High CodeQL alert: $high_alert"
    fi
done <<<"$high_alerts"

if [ -n "$critical_alerts" ]; then
    echo "Open critical CodeQL alerts stop this release. Fix them, or dismiss them with a reason and rerun this job:"
    echo "$critical_alerts"
    exit 1
fi

echo "No open critical CodeQL alert in the release analysis"
