@extends("App::app")
@section('title','Logger分享')
@section('content')

    <div class="row row-cards">
        <div class="border-0 card">
            <div class="card-header">
                <div class="col-auto"><h3 class="card-title">日志信息分享</h3></div>

            </div>
            <div class="card-body">
<pre><code>@json([$data],JSON_UNESCAPED_UNICODE|JSON_PRETTY_PRINT)</code>
</pre>
            </div>
        </div>
    </div>

@endsection