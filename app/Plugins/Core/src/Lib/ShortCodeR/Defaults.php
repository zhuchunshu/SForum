<?php

namespace App\Plugins\Core\src\Lib\ShortCodeR;

use App\Plugins\Comment\src\Model\TopicComment;
use App\Plugins\Topic\src\Models\Topic;
use Hyperf\Utils\Str;
use Thunder\Shortcode\Shortcode\ShortcodeInterface;
class Defaults
{
    public function alert_success($match, ShortcodeInterface $s)
    {
        return <<<HTML
<div class="alert alert-important alert-success alert-dismissible">
  <div class="d-flex">
    <div class="shortcode-alert-icon">
      <!-- Download SVG icon from http://tabler-icons.io/i/check -->
      <svg
        xmlns="http://www.w3.org/2000/svg"
        class="icon alert-icon"
        width="24"
        height="24"
        viewBox="0 0 24 24"
        stroke-width="2"
        stroke="currentColor"
        fill="none"
        stroke-linecap="round"
        stroke-linejoin="round"
      >
        <path stroke="none" d="M0 0h24v24H0z" fill="none" />
        <path d="M5 12l5 5l10 -10" />
      </svg>
    </div>
    <div class="shortcode-alert-text">{$s->getContent()}</div>
  </div>
</div>
HTML;
    }
    public function alert_error($match, ShortcodeInterface $s)
    {
        return <<<HTML
<div class="alert alert-important alert-danger alert-dismissible">
  <div class="d-flex">
    <div class="shortcode-alert-icon">
      <!-- Download SVG icon from http://tabler-icons.io/i/check -->
      <svg xmlns="http://www.w3.org/2000/svg" class="icon alert-icon" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path stroke="none" d="M0 0h24v24H0z" fill="none"></path><circle cx="12" cy="12" r="9"></circle><line x1="12" y1="8" x2="12" y2="12"></line><line x1="12" y1="16" x2="12.01" y2="16"></line></svg>
    </div>
    <div class="shortcode-alert-text">{$s->getContent()}</div>
  </div>
</div>

HTML;
    }
    public function alert_info($match, ShortcodeInterface $s)
    {
        return <<<HTML
<div class="alert alert-important alert-info alert-dismissible">
  <div class="d-flex">
    <div class="shortcode-alert-icon">
      <svg
        xmlns="http://www.w3.org/2000/svg"
        class="icon alert-icon"
        width="24"
        height="24"
        viewBox="0 0 24 24"
        stroke-width="2"
        stroke="currentColor"
        fill="none"
        stroke-linecap="round"
        stroke-linejoin="round"
      >
        <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
        <circle cx="12" cy="12" r="9"></circle>
        <line x1="12" y1="8" x2="12.01" y2="8"></line>
        <polyline points="11 12 12 12 12 16 13 16"></polyline>
      </svg>
    </div>
    <div class="shortcode-alert-text">{$s->getContent()}</div>
  </div>
</div>

HTML;
    }
    public function alert_warning($match, ShortcodeInterface $s)
    {
        return <<<HTML
<div class="alert alert-important alert-warning alert-dismissible">
  <div class="d-flex">
    <div class="shortcode-alert-icon">
    <svg xmlns="http://www.w3.org/2000/svg" class="icon alert-icon" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path stroke="none" d="M0 0h24v24H0z" fill="none"></path><path d="M12 9v2m0 4v.01"></path><path d="M5 19h14a2 2 0 0 0 1.84 -2.75l-7.1 -12.25a2 2 0 0 0 -3.5 0l-7.1 12.25a2 2 0 0 0 1.75 2.75"></path></svg>
    </div>
    <div class="shortcode-alert-text">{$s->getContent()}</div>
  </div>
</div>

HTML;
    }
    public function topic($match, ShortcodeInterface $s)
    {
        $topic_id = $s->getParameter('topic_id');
        if (!Topic::query()->where("id", $topic_id)->exists()) {
            return '[' . $s->getName() . '] ' . __("app.Error using short tags");
        }
        return <<<HTML
<div class="card border-1 hvr-grow row topic-with" core-data="topic" topic-id="{$topic_id}">
    <div class="col" core-data="topic_content">
        <div class="skeleton-line"></div>
        <div class="skeleton-line"></div>
        <div class="skeleton-line"></div>
        <div class="skeleton-line"></div>
    </div>
    <div class="col-auto" core-data="topic-author">
        <div class="skeleton-avatar skeleton-avatar-md"></div>
    </div>
</div>
HTML;
    }
    public function chart($match, ShortcodeInterface $s)
    {
        $data = strip_tags($s->getContent());
        $id = "chart-" . Str::random();
        return <<<HTML
<div style="padding: 1rem 1rem;">
    <div id="{$id}"></div>
</div>
<script>
      // @formatter:off
      document.addEventListener("DOMContentLoaded", function () {
      \twindow.ApexCharts && (new ApexCharts(document.getElementById('{$id}'), {$data})).render();
      });
      // @formatter:on
   </script>
HTML;
    }
    public function button($match, ShortcodeInterface $s)
    {
        return <<<HTML
    <a href="{$s->getParameter('url', '/')}" class="btn {$s->getParameter('class', 'btn-light')}">{$s->getContent()}</a>
HTML;
    }
    public function file($match, ShortcodeInterface $s)
    {
        $name = $s->getParameter('name');
        $url = $s->getParameter('url');
        $pwd = $s->getParameter('password');
        $unzip = $s->getParameter('unzip');
        $pwd ?: ($pwd = "无");
        $unzip ?: ($unzip = "无");
        if (!@$name || !@$url) {
            return <<<HTML
<div class="alert alert-danger" role="alert">
  <b class="alert-title">附件加载失败&hellip;</b>
  <div class="text-muted">您创建的附件因参数不足，所以无法加载</div>
</div>
HTML;
        }
        return <<<HTML
<div class="alert alert-success alert-dismissible" role="alert">
  <b class="mb-1"><svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-cloud-download" width="30" height="30" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M19 18a3.5 3.5 0 0 0 0 -7h-1a5 4.5 0 0 0 -11 -2a4.6 4.4 0 0 0 -2.1 8.4"></path>
   <line x1="12" y1="13" x2="12" y2="22"></line>
   <polyline points="9 19 12 22 15 19"></polyline>
</svg>{$name}</b>
  <p>
  <ul>
  <li>提取码:{$pwd}</li>
  <li>解压码:{$unzip}</li>
</ul>
</p>
  <div class="btn-list">
    <a href="{$url}" class="btn btn-success">下载</a>
  </div>
</div>
HTML;
    }
}