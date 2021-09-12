<?php


namespace App\Plugins\Core\src\Lib\ShortCodeR;


use App\Plugins\Topic\src\Models\Topic;

class Defaults
{
  public static function a($match)
  {
    return "<a>$match[1]</a>";
  }

  public function alert_success($match)
  {
    return <<<HTML
<div class="alert alert-important alert-success alert-dismissible">
  <div class="d-flex">
    <div>
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
    <div>{$match[1]}</div>
  </div>
</div>
HTML;
  }

  public function alert_error($match)
  {
    return <<<HTML
<div class="alert alert-important alert-danger alert-dismissible">
  <div class="d-flex">
    <div>
      <!-- Download SVG icon from http://tabler-icons.io/i/check -->
      <svg xmlns="http://www.w3.org/2000/svg" class="icon alert-icon" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path stroke="none" d="M0 0h24v24H0z" fill="none"></path><circle cx="12" cy="12" r="9"></circle><line x1="12" y1="8" x2="12" y2="12"></line><line x1="12" y1="16" x2="12.01" y2="16"></line></svg>
    </div>
    <div>{$match[1]}</div>
  </div>
</div>

HTML;
  }

  public function alert_info($match)
  {
    return <<<HTML
<div class="alert alert-important alert-info alert-dismissible">
  <div class="d-flex">
    <div>
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
    <div>{$match[1]}</div>
  </div>
</div>

HTML;
  }

  public function alert_warning($match)
  {
    return <<<HTML
<div class="alert alert-important alert-warning alert-dismissible">
  <div class="d-flex">
    <div>
    <svg xmlns="http://www.w3.org/2000/svg" class="icon alert-icon" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path stroke="none" d="M0 0h24v24H0z" fill="none"></path><path d="M12 9v2m0 4v.01"></path><path d="M5 19h14a2 2 0 0 0 1.84 -2.75l-7.1 -12.25a2 2 0 0 0 -3.5 0l-7.1 12.25a2 2 0 0 0 1.75 2.75"></path></svg>
    </div>
    <div>{$match[1]}</div>
  </div>
</div>

HTML;
  }
  public function topic($match){
      $topic_id = $match[1];
      if(!Topic::query()->where("id",$topic_id)->exists()) {
          return '[topic id="'.$topic_id.'"][/topic]';
      }
      $data = Topic::query()->where("id",$topic_id)->select("id","title","user_id","options","created_at")->with("user")->first();
      $user_avatar = super_avatar($data->user);
      $title = \Hyperf\Utils\Str::limit($data->title,20);
      $summary = \Hyperf\Utils\Str::limit(core_default(deOptions($data->options)["summary"],"未捕获到本文摘要"),40);
      return <<<HTML
<div class="row topic-with">
    <div data-bs-toggle="tooltip" data-bs-placement="top" title="引用的帖子" class="col">
        <a href="/{$data->id}.html" class="text-reset" style="text-decoration:none;"><b>{$title}</b></a>
        <a href="/{$data->id}.html" style="display: -webkit-box;
    font-size: 13px;
    height: 18px;
    line-height: 18px;
    color: #999999;
    word-break: break-all;
    text-overflow: ellipsis;
    overflow: hidden;
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 1;
    text-decoration:none;">
            {$summary}
        </a>
    </div>
    <div class="col-auto">
    <a href="/users/{$data->user->username}.html" class="avatar" style="background-image: url($user_avatar)"></a>
    </div>
</div>
HTML;

  }
}
