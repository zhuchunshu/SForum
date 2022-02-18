@extends('app')

@section('title','Admin')

@section('content')
    <div class="row row-cards">
        <div class="col-md-12" id="vue-admin-index-releases">
            <div class="row row-cards">
                <div class="col-md-12" v-if="data && commit">
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
                                        <a :href="data.tarball_url" class="btn btn-light">下载tar.gz包</a>
                                    </div>
                                </div>
                            </div>
                        </div>

{{--                        commit --}}
                        <div class="col-md-6">
                            <div class="border-0 card" v-if="commit">
                                <div class="card-body">
                                    <h3 class="card-title">Commit</h3>
                                    <div style="overflow:scroll; overflow-x:hidden;max-height: 400px;width:100%">
                                        <ul class="list list-timeline list-timeline-simple">
                                            <li v-for="value in commit">
                                                <a v-if="value.author" :href="value.author.html_url" class="list-timeline-icon bg-twitter">
                                                    <img :src="value.author.avatar_url" alt="">
                                                </a>
                                                <div v-if="value.commit" class="list-timeline-content">
                                                    <div v-if="value.commit.author" class="list-timeline-time">@{{ value.commit.author.date }}</div>
                                                    <p v-if="value.commit.author" class="list-timeline-title">
                                                        <a :href="value.html_url">@{{ value.commit.author.name }}</a>
                                                    </p>
                                                    <a class="text-muted" :href="value.html_url" v-html="value.commit.message"></a>
                                                </div>
                                            </li>
                                        </ul>
                                    </div>
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