@extends('app')

@section('title','MeiliSearch')

@section('content')

    <div class="col-md-6">

        <div class="card">
            <div class="card-body">
                <h3 class="card-title">
                    MeiliSearch 设置
                </h3>
                <form action="" method="post">
                    <x-csrf/>
                    <div class="mb-3">
                        <label for="" class="form-label">meilisearch 连接地址</label>
                        <input type="url" class="form-control" name="url" placeholder="" value="{{get_options("meilisearch_url",'http://127.0.0.1:7700')}}">
                    </div>
                    <div class="mb-3">
                        <label for="" class="form-label">meilisearch apikey</label>
                        <input type="text" class="form-control" name="apikey"  placeholder="" value="{{get_options("meilisearch_apikey",null)}}">
                    </div>
                    <div class="mb-3">
                        <label for="" class="form-label">索引名</label>
                        <input type="text" class="form-control" name="index"  placeholder="" value="{{get_options("meilisearch_index",get_options("APP_NAME","SuperForum"))}}">
                    </div>
                    <button class="btn btn-primary">保存</button>
                </form>
            </div>
        </div>

    </div>

    <div class="col-md-6">
        <div class="card">
            <div class="card-body">
                <h3 class="card-title">状态</h3>
                @if($code===200 && $status ==='available')<span class="status status-green">
  <span class="status-dot status-dot-animated"></span>
  {{$status}}
</span>@else
                    <span class="status status-red">
  <span class="status-dot status-dot-animated"></span>
  {{$status}}
</span>
                @endif
            </div>
        </div>
    </div>

@endsection