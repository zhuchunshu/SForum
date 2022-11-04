@extends('app')

@section('title','Logger')

@section('content')
    <div class="border-0 card">
        <div class="card-header">
            <div class="col-auto"><h3 class="card-title">日志信息查看</h3></div>
            <div class="card-actions">
                <div class="col"><a href="/admin/server/logger" class="btn">Logger列表</a></div>
            </div>
        </div>
        <div class="card-body">
<pre><code>@json([$data],JSON_UNESCAPED_UNICODE)</code>
</pre>
        </div>
        <div class="card-footer">
            <h3 class="card-title">分享链接：</h3>
            <div class="input-icon">
                <input type="text" value="{{url('/share/admin/server/logger/'.$_token.'.debug')}}" class="form-control" placeholder="Search…" readonly>
            </div>
        </div>
    </div>
@endsection