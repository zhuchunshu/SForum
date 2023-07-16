<?php

use Hyperf\Database\Schema\Schema;
use Hyperf\Database\Schema\Blueprint;
use Hyperf\Database\Migrations\Migration;

class CreateTopicCommentTable extends Migration
{
    /**
     * Run the migrations.
     */
    public function up(): void
    {
        Schema::create('topic_comment', function (Blueprint $table) {
            $table->bigIncrements('id');
            $table->integer('likes')->default(0);
            $table->string('topic_id');
            $table->string('user_id');
            $table->string('parent_id')->nullable();
            $table->longText("content");
            $table->longText("markdown")->nullable();
            $table->string("status")->default('publish');
            $table->timestamp("shenping")->comment("神评")->nullable();
            $table->timestamp("optimal")->comment("最佳回复")->nullable();
            $table->timestamps();
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::dropIfExists('topic_comment');
    }
}
