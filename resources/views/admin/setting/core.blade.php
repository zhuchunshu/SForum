@extends("app")

@section('title',"站点设置")

@section('content')
<div class="col-md-12">
    <div class="row row-cards" id="setting-core-form">
    <div class="col-md-12">
        <!-- Cards with tabs component -->
        <div class="card-tabs border-0">
            <!-- Cards navigation -->
            <ul class="nav nav-tabs">
                <li class="nav-item"><a href="#default" class="nav-link active" data-bs-toggle="tab">基本设置</a></li>
            </ul>
            <div class="tab-content" id="setting-core-form">
                <!-- Content of card #1 -->
                @include('admin.setting.core.1')
            </div>
        </div>
    </div>
    <div class="col-md-12">
        <button class="btn btn-primary">提交</button>
    </div>
    </div>
</div>
@endsection

@section('scripts')
    <script src="{{ mix("js/admin/setting.js") }}"></script>
@endsection