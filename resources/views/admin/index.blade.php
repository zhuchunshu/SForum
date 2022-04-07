@extends('app')

@section('title','Admin')

@section('content')
    <div class="row row-cards">
        <div class="col-md-12" id="vue-admin-index-releases">
            <div class="row row-cards">
                <div class="col-md-12" v-if="data">
                    <div class="row row-cards">

{{--                        当前版本--}}
                        <div class="col-md-6">
                            <div class="border-0 card" v-if="data">
                                <div class="card-body">
                                    <h3 class="card-title">Releases</h3>
                                    <p>当前版本:@{{data.version}}</p>
                                    <p>最新版本: <a :href="data.new_version_url">@{{data.tag_name}}</a></p>
                                    <div v-if="data.upgrade">
                                        可更新
                                        <br>
                                        <a :href="data.zipball_url" class="btn btn-dark" style="margin-right: 10px">下载zip包</a>
                                        <a :href="data.tarball_url" style="margin-right: 10px" class="btn btn-light">下载tar.gz包</a>
                                        <button @@click="update" class="btn btn-green">立即更新</button>
                                    </div>
                                </div>
                                <div class="card-footer">
                                    <div class="col-md-12">
                                        <button @@click="clearCache" class="btn btn-dark">清理缓存</button>
                                    </div>
                                </div>
                            </div>
                        </div>
                        <div class="col-md-6">
                            <div class="border-0 card" v-if="data">
                                <div class="card-body">
                                    <h3 class="card-title">其他信息</h3>
                                    <p>官网: <a href="https://www.runpod.cn">https://www.runpod.cn</a> </p>
                                    <p>文档: <a href="https://www.runpod.cn/docs">https://www.runpod.cn/docs</a> </p>\
                                    <p>开源地址: <a href="https://github.com/zhuchunshu/super-forum">https://github.com/zhuchunshu/super-forum</a> </p>
                                </div>
                            </div>
                        </div>

{{--                        commit --}}
                        <div class="col-md-12">
                            <div class="border-0 card" v-if="markdown">
                                <div class="card-body">
                                    <h3 class="card-title">
                                        <div class="row">
                                            <div class="col">更新日志</div>
                                            <div class="col-auto"><a href="https://forum.runpod.cn/48.html">查看</a> </div>
                                        </div>
                                    </h3>
                                    <div style="overflow:scroll;overflow-x:hidden;max-height: 700px;" class="markdown" v-html="updateLog"></div>
                                </div>
                            </div>
                        </div>


                    </div>
                </div>

{{--                加载中--}}
                <div class="col-md-12" v-else>
                    <div class="border-0 card">
                        <div class="card-body">
                            <div class="empty">
                                <p class="empty-title">Loadding...</p>

                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </div>
@endsection

@section('scripts')
    <script src="{{mix('js/admin/index.js')}}"></script>
@endsection