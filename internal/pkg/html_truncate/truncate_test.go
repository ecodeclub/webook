package html_truncate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPartService(t *testing.T) {
	testcases := []struct {
		name        string
		content     string
		number      int
		wantContent string
	}{
		{
			name:        "2个段落",
			content:     `<p>Paragraph 1</p><p>Paragraph 2</p><p>Paragraph 3</p>`,
			number:      2,
			wantContent: `<p>Paragraph 1</p><p>Paragraph 2</p>`,
		},
		{
			name:        "超出拥有的段落",
			content:     `<p>Paragraph 1</p><p>Paragraph 2</p>`,
			number:      5,
			wantContent: `<p>Paragraph 1</p><p>Paragraph 2</p>`,
		},
		{
			name:        "0个段落",
			content:     `<p>Paragraph 1</p><p>Paragraph 2</p>`,
			number:      0,
			wantContent: ``,
		},
		{
			name:        "空字符串",
			content:     ``,
			number:      0,
			wantContent: ``,
		},
		{
			name:        "包含其他标签截取1段",
			content:     `<p>在微服务架构中，服务实例的动态变化是常态。为了保障系统的稳定性和可用性，需要构建一套完善的机制来处理这些变化。具体可以从以下三个方面入手：</p><p></p><ol><li>服务注册中心的动态管理：服务注册中心是处理服务实例动态变化的核心组件，其关键机制包括心跳检测和实例状态同步：<ul><li>服务注册与注销：服务实例上线后主动注册到注册中心；服务实例下线时主动注销，避免无效调用。</li><li>心跳检测：注册中心定期检测实例的健康状态，超时未响应则移除实例。</li><li>实例状态同步：注册中心将实例状态变化（上线、下线、故障）实时同步给消费者。同步方式包括推送和拉取，大规模场景下可采用增量同步、服务注册中心分区或分层架构等优化策略。</li></ul></li><li>服务消费者侧的容错策略：服务消费者侧需要一系列容错策略来应对服务实例的动态变化：<ul><li>实例缓存与动态更新：服务消费者缓存实例列表，并通过注册中心的推送或定期拉取机制更新缓存。</li><li>负载均衡：采用负载均衡策略（如轮询、加权轮询、一致性哈希）分发流量，并根据实例的健康状态动态调整权重。一致性哈希可以减少实例变化带来的缓存失效。</li><li>容错机制：包括重试机制、熔断机制、降级处理和限流保护等，确保在部分实例不可用时，服务仍然可以正常运行。熔断机制的核心是在连续失败达到一定阈值后，短时间内停止对故障实例的调用，避免级联故障。</li></ul></li><li>动态扩容与缩容的平滑处理：<ul><li>扩容：新服务实例上线后，注册中心自动注册并同步给消费者。负载均衡组件逐步增加新实例的权重，实现流量的平滑过渡。</li><li>缩容：服务实例下线前，逐步减少实例的权重，并处理完已有请求后再注销，避免流量损失。</li></ul></li></ol><p>总而言之，微服务架构中，处理服务实例动态变化的核心在于服务注册中心与服务消费者的协同配合。注册中心通过心跳检测、健康检查和实例状态同步等机制感知实例变化，消费者通过负载均衡和容错策略应对这些变化，共同保障系统的稳定性和高可用性。</p>`,
			number:      1,
			wantContent: `<p>在微服务架构中，服务实例的动态变化是常态。为了保障系统的稳定性和可用性，需要构建一套完善的机制来处理这些变化。具体可以从以下三个方面入手：</p>`,
		},
		{
			name:        "包含其他标签截取2段",
			content:     `<p>在微服务架构中，服务实例的动态变化是常态。为了保障系统的稳定性和可用性，需要构建一套完善的机制来处理这些变化。具体可以从以下三个方面入手：</p><p></p><ol><li>服务注册中心的动态管理：服务注册中心是处理服务实例动态变化的核心组件，其关键机制包括心跳检测和实例状态同步：<ul><li>服务注册与注销：服务实例上线后主动注册到注册中心；服务实例下线时主动注销，避免无效调用。</li><li>心跳检测：注册中心定期检测实例的健康状态，超时未响应则移除实例。</li><li>实例状态同步：注册中心将实例状态变化（上线、下线、故障）实时同步给消费者。同步方式包括推送和拉取，大规模场景下可采用增量同步、服务注册中心分区或分层架构等优化策略。</li></ul></li><li>服务消费者侧的容错策略：服务消费者侧需要一系列容错策略来应对服务实例的动态变化：<ul><li>实例缓存与动态更新：服务消费者缓存实例列表，并通过注册中心的推送或定期拉取机制更新缓存。</li><li>负载均衡：采用负载均衡策略（如轮询、加权轮询、一致性哈希）分发流量，并根据实例的健康状态动态调整权重。一致性哈希可以减少实例变化带来的缓存失效。</li><li>容错机制：包括重试机制、熔断机制、降级处理和限流保护等，确保在部分实例不可用时，服务仍然可以正常运行。熔断机制的核心是在连续失败达到一定阈值后，短时间内停止对故障实例的调用，避免级联故障。</li></ul></li><li>动态扩容与缩容的平滑处理：<ul><li>扩容：新服务实例上线后，注册中心自动注册并同步给消费者。负载均衡组件逐步增加新实例的权重，实现流量的平滑过渡。</li><li>缩容：服务实例下线前，逐步减少实例的权重，并处理完已有请求后再注销，避免流量损失。</li></ul></li></ol><p>总而言之，微服务架构中，处理服务实例动态变化的核心在于服务注册中心与服务消费者的协同配合。注册中心通过心跳检测、健康检查和实例状态同步等机制感知实例变化，消费者通过负载均衡和容错策略应对这些变化，共同保障系统的稳定性和高可用性。</p>`,
			number:      2,
			wantContent: `<p>在微服务架构中，服务实例的动态变化是常态。为了保障系统的稳定性和可用性，需要构建一套完善的机制来处理这些变化。具体可以从以下三个方面入手：</p><p></p><ol><li>服务注册中心的动态管理：服务注册中心是处理服务实例动态变化的核心组件，其关键机制包括心跳检测和实例状态同步：<ul><li>服务注册与注销：服务实例上线后主动注册到注册中心；服务实例下线时主动注销，避免无效调用。</li><li>心跳检测：注册中心定期检测实例的健康状态，超时未响应则移除实例。</li><li>实例状态同步：注册中心将实例状态变化（上线、下线、故障）实时同步给消费者。同步方式包括推送和拉取，大规模场景下可采用增量同步、服务注册中心分区或分层架构等优化策略。</li></ul></li><li>服务消费者侧的容错策略：服务消费者侧需要一系列容错策略来应对服务实例的动态变化：<ul><li>实例缓存与动态更新：服务消费者缓存实例列表，并通过注册中心的推送或定期拉取机制更新缓存。</li><li>负载均衡：采用负载均衡策略（如轮询、加权轮询、一致性哈希）分发流量，并根据实例的健康状态动态调整权重。一致性哈希可以减少实例变化带来的缓存失效。</li><li>容错机制：包括重试机制、熔断机制、降级处理和限流保护等，确保在部分实例不可用时，服务仍然可以正常运行。熔断机制的核心是在连续失败达到一定阈值后，短时间内停止对故障实例的调用，避免级联故障。</li></ul></li><li>动态扩容与缩容的平滑处理：<ul><li>扩容：新服务实例上线后，注册中心自动注册并同步给消费者。负载均衡组件逐步增加新实例的权重，实现流量的平滑过渡。</li><li>缩容：服务实例下线前，逐步减少实例的权重，并处理完已有请求后再注销，避免流量损失。</li></ul></li></ol><p>总而言之，微服务架构中，处理服务实例动态变化的核心在于服务注册中心与服务消费者的协同配合。注册中心通过心跳检测、健康检查和实例状态同步等机制感知实例变化，消费者通过负载均衡和容错策略应对这些变化，共同保障系统的稳定性和高可用性。</p>`,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			h := htmlTruncator{}
			gotContent := h.TruncateByParagraphs(tc.content, tc.number)
			assert.Equal(t, tc.wantContent, gotContent)
		})
	}
}

func TestParagraphCount(t *testing.T) {
	testcases := []struct {
		name    string
		content string
		want    int
	}{
		{
			name:    "3个段落",
			content: `<p>Paragraph 1</p><p>Paragraph 2</p><p>Paragraph 3</p>`,
			want:    3,
		},
		{
			name:    "2个段落和1个空段落",
			content: `<p>Paragraph 1</p><p></p><p>Paragraph 2</p>`,
			want:    2,
		},
		{
			name:    "0个段落",
			content: ``,
			want:    0,
		},
		{
			name:    "只有空段落",
			content: `<p></p><p></p>`,
			want:    0,
		},
		{
			name:    "包含其他标签",
			content: `<p>Paragraph 1</p><div>Some content</div><p>Paragraph 2</p>`,
			want:    2,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			h := htmlTruncator{}
			got := h.ParagraphCount(tc.content)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestTruncate(t *testing.T) {
	testcases := []struct {
		name        string
		content     string
		wantContent string
	}{
		{
			name:        "少于等于3个段落",
			content:     `<p>Paragraph 1</p><p>Paragraph 2</p><p>Paragraph 3</p>`,
			wantContent: `<p>Paragraph 1</p>`,
		},
		{
			name:        "正好4个段落",
			content:     `<p>Paragraph 1</p><p>Paragraph 2</p><p>Paragraph 3</p><p>Paragraph 4</p>`,
			wantContent: `<p>Paragraph 1</p><p>Paragraph 2</p>`,
		},
		{
			name:        "超过4个段落",
			content:     `<p>Paragraph 1</p><p>Paragraph 2</p><p>Paragraph 3</p><p>Paragraph 4</p><p>Paragraph 5</p>`,
			wantContent: `<p>Paragraph 1</p><p>Paragraph 2</p><p>Paragraph 3</p>`,
		},
		{
			name:        "没有段落",
			content:     ``,
			wantContent: ``,
		},
		{
			name:        "包含空段落",
			content:     `<p>Paragraph 1</p><p></p><p>Paragraph 2</p><p>Paragraph 3</p>`,
			wantContent: `<p>Paragraph 1</p>`,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			h := htmlTruncator{}
			gotContent := h.Truncate(tc.content)
			assert.Equal(t, tc.wantContent, gotContent)
		})
	}
}
