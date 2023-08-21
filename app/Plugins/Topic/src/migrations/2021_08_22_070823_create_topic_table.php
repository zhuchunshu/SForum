<?php

use Hyperf\Database\Schema\Schema;
use Hyperf\Database\Schema\Blueprint;
use Hyperf\Database\Migrations\Migration;

class CreateTopicTable extends Migration
{
    /**
     * Run the migrations.
     */
    public function up(): void
    {
        Schema::create('topic', function (Blueprint $table) {
            $table->bigIncrements('id');
            $table->text('title');
            $table->string('user_id')->comment('作者id');
            $table->string('status')->comment('状态');
            $table->longText("content")->comment('内容');
            $table->longText("markdown")->comment('markdown内容');
            $table->integer('like')->comment('点赞数量')->default(0)->nullable();
            $table->integer('view')->comment('浏览量')->default(0)->nullable();
            $table->string('tag_id')->comment('板块id')->default(1);
            $table->string('_token');
            $table->text('options')->nullable();
            $table->timestamps();
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('topic');
    }
}
